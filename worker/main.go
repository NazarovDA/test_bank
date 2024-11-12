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
	FromAccountID int     `json:"from_account_id"`
	ToAccountID   int     `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

func main() {
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqQueue := os.Getenv("RABBITMQ_QUEUE")
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
		}

		log.Printf("Transaction processed: %+v", req)
	}
}

func processTransaction(db *sql.DB, req TransactionRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return logTransactionError(db, req, "failed", "failed to begin transaction")
	}

	var fromBalance float64
	err = tx.QueryRow("SELECT balance FROM clients WHERE id = $1", req.FromAccountID).Scan(&fromBalance)
	if err != nil {
		tx.Rollback()
		return logTransactionError(db, req, "failed", "sender not found or query error")
	}
	if fromBalance < req.Amount {
		tx.Rollback()
		return logTransactionError(db, req, "failed", "insufficient funds")
	}

	_, err = tx.Exec("UPDATE clients SET balance = balance - $1 WHERE id = $2", req.Amount, req.FromAccountID)
	if err != nil {
		tx.Rollback()
		return logTransactionError(db, req, "failed", "failed to debit sender's account")
	}
	_, err = tx.Exec("UPDATE clients SET balance = balance + $1 WHERE id = $2", req.Amount, req.ToAccountID)
	if err != nil {
		tx.Rollback()
		return logTransactionError(db, req, "failed", "failed to credit recipient's account")
	}

	_, err = tx.Exec("INSERT INTO transactions (from_account_id, to_account_id, amount, status) VALUES ($1, $2, $3, $4)",
		req.FromAccountID, req.ToAccountID, req.Amount, "completed")
	if err != nil {
		tx.Rollback()
		return logTransactionError(db, req, "failed", "failed to log completed transaction")
	}

	return tx.Commit()
}

func logTransactionError(db *sql.DB, req TransactionRequest, status, errorMsg string) error {
	_, err := db.Exec("INSERT INTO transactions (from_account_id, to_account_id, amount, status, error_message) VALUES ($1, $2, $3, $4, $5)",
		req.FromAccountID, req.ToAccountID, req.Amount, status, errorMsg)
	if err != nil {
		log.Printf("Failed to log transaction error: %v", err)
	}
	return fmt.Errorf("%s: %s", status, errorMsg)
}
