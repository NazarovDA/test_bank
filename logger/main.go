package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

type TransActionLog struct {
	TransactionID string `json:"id,omitempty"`
	SenderId      int    `json:"sender"`
	ReceiverId    int    `json:"receiver"`
	Amount        int    `json:"amount"`
	Status        string `json:"status" default:"pending"`
}

func main() {
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqQueue := os.Getenv("RABBITMQ_LOG_QUEUE")

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

	channel, err := conn.Channel()
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

	log.Print("Logger is ready")

	for msg := range msgs {
		log.Printf("Message received: %v", msg)

		var req TransActionLog

		if err := json.Unmarshal(msg.Body, &req); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		if err := processLog(db, req); err != nil {
			log.Printf("Failed to process log: %v", err)
			continue
		}

		log.Printf("Log processed: %+v", req)
	}
}

func processLog(db *sql.DB, req TransActionLog) error {
	tx, err := db.Begin()

	if err != nil {
		log.Println(87)
		return err
	}

	var status string
	err = tx.QueryRow("SELECT status from transactions where id = $1", req.TransactionID).Scan(&status)

	if err != nil && err == sql.ErrNoRows {
		tx.Rollback()
		log.Println(96)
		return err
	}
	if status == "successful" {
		log.Printf("Transaction is already successful")
	}

	_, err = tx.Exec(
		"INSERT INTO transactions (id, from_account_id, to_account_id, amount, status) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(id) DO UPDATE SET status = $5",
		req.TransactionID, req.SenderId, req.ReceiverId, req.Amount, req.Status,
	)

	if err != nil {
		tx.Rollback()
		log.Println(107)
		return err
	}

	return tx.Commit()
}
