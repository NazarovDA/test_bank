package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

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

var db *sql.DB

func main() {
	go func() {
		http.HandleFunc("/isalive", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		port := os.Getenv("LOGGER_PORT")
		if port == "" {
			port = "8080"
		}
		log.Print(port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	go func() {
		rabbitmqHost := os.Getenv("RABBITMQ_HOST")
		rabbitmqQueue := os.Getenv("RABBITMQ_LOG_QUEUE")

		dbHost := os.Getenv("DB_HOST")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")

		connStr := "host=" + dbHost + " user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"
		var err error
		db, err = sql.Open("postgres", connStr)
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

		if _, err = channel.QueueDeclare(
			rabbitmqQueue, // routing key
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			log.Fatalf("Failed to declare a queue: %v", err)
		} else {
			log.Print("Queue declared")
		}

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
	}()

	go func() {
		for {
			if db != nil && db.Ping() != nil {
				log.Fatal("Failed to ping database")
				time.Sleep(5 * time.Second)
				os.Exit(1)
			}
		}
	}()

	select {}
}

func processLog(db *sql.DB, req TransActionLog) error {
	tx, err := db.Begin()

	if err != nil {
		log.Println(87)
		return err
	}

	var status string
	err = tx.QueryRow("SELECT status from transactions where id = $1", req.TransactionID).Scan(&status)

	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}
	if status == "successful" {
		log.Printf("Transaction is already successful")
		return nil
	}

	_, err = tx.Exec(
		"INSERT INTO transactions (id, from_account_id, to_account_id, amount, status) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(id) DO UPDATE SET status = $5",
		req.TransactionID, req.SenderId, req.ReceiverId, req.Amount, req.Status,
	)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
