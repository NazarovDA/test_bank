package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

type TransactionRequest struct {
	TransactionID string  `json:"id"`
	FromAccountID int     `json:"from_account_id"`
	ToAccountID   int     `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}
type TransActionLog struct {
	TransactionID string  `json:"id"`
	SenderId      int     `json:"sender"`
	ReceiverId    int     `json:"receiver"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status" default:"pending"`
}

var channel *amqp.Channel
var rabbitmqLogQueue string

func main() {
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqQueue := os.Getenv("RABBITMQ_TRANSACTION_QUEUE")
	rabbitmqLogQueue = os.Getenv("RABBITMQ_LOG_QUEUE")
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := "host=" + dbHost + " user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	conn, err := amqp.Dial("amqp://" + rabbitmqHost)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	channel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer channel.Close()

	msgs, err := channel.Consume(
		rabbitmqQueue, // queue
		"",            // consumer
		true,          // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for msg := range msgs {
		var req TransactionRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		err := processTransaction(db, req)
		if err != nil {
			log.Printf("Failed to process transaction: %v", err)
			continue
		} else {

		}

		log.Printf("Transaction processed: %+v", req)
	}
}

func sendLog(req TransactionRequest, status string) {
	logReq := TransActionLog{
		TransactionID: req.TransactionID,
		SenderId:      req.FromAccountID,
		ReceiverId:    req.ToAccountID,
		Amount:        req.Amount,
		Status:        status,
	}
	logBody, err := json.Marshal(logReq)
	if err != nil {
		return
	}
	_ = channel.Publish(
		"",               // exchange
		rabbitmqLogQueue, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         logBody,
			DeliveryMode: amqp.Persistent,
		})
}

func processTransaction(db *sql.DB, req TransactionRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var fromBalance float64
	err = tx.QueryRow("SELECT balance FROM clients WHERE id = $1", req.FromAccountID).Scan(&fromBalance)
	if err != nil {
		tx.Rollback()
		sendLog(req, err.Error())
		return err
	}
	if fromBalance < req.Amount {
		tx.Rollback()
		err = fmt.Errorf("not enough money")
		sendLog(req, err.Error())
		return err
	}

	_, err = tx.Exec("UPDATE clients SET balance = balance - $1 WHERE id = $2", req.Amount, req.FromAccountID)
	if err != nil {
		tx.Rollback()
		sendLog(req, err.Error())
		return err
	}
	_, err = tx.Exec("UPDATE clients SET balance = balance + $1 WHERE id = $2", req.Amount, req.ToAccountID)
	if err != nil {
		tx.Rollback()
		sendLog(req, err.Error())
		return err
	}

	sendLog(req, "successful")

	return tx.Commit()
}
