package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

func createUuid() string {
	return uuid.New().String()
}

type TransactionRequest struct {
	TransactionID string  `json:"id"`
	FromAccountID int     `json:"from_account_id"`
	ToAccountID   int     `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}
type TransactionLog struct {
	TransactionID string  `json:"id"`
	SenderId      int     `json:"sender"`
	ReceiverId    int     `json:"receiver"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status" default:"pending"`
}

func main() {
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqQueue := os.Getenv("RABBITMQ_TRANSACTION_QUEUE")
	rabbitmqLogQueue := os.Getenv("RABBITMQ_LOG_QUEUE")

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

	if _, err = channel.QueueDeclare(
		rabbitmqLogQueue, // routing key
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

	http.HandleFunc("POST /transfer", func(w http.ResponseWriter, r *http.Request) {
		var req TransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}
		req.TransactionID = createUuid()
		body, err := json.Marshal(req)
		if err != nil {
			http.Error(w, "Failed to process request", http.StatusInternalServerError)
			return
		}

		logReq := TransactionLog{
			TransactionID: req.TransactionID,
			SenderId:      req.FromAccountID,
			ReceiverId:    req.ToAccountID,
			Amount:        req.Amount,
			Status:        "Pending",
		}
		logBody, err := json.Marshal(logReq)
		if err != nil {
			http.Error(w, "Failed to process request", http.StatusInternalServerError)
			return
		}
		err = channel.Publish(
			"",               // exchange
			rabbitmqLogQueue, // routing key
			false,            // mandatory
			false,            // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         logBody,
				DeliveryMode: amqp.Persistent,
			})

		if err != nil {
			http.Error(w, "Failed to queue request", http.StatusInternalServerError)
			return
		}

		log.Printf("Message prepared: %s", body)
		err = channel.Publish(
			"",            // exchange
			rabbitmqQueue, // routing key
			false,         // mandatory
			false,         // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				DeliveryMode: amqp.Persistent,
			})
		if err != nil {
			http.Error(w, "Failed to queue request", http.StatusInternalServerError)
			return
		} else {
			log.Printf("Message sent: %s", body)
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Transaction queued successfully"))
	})

	http.HandleFunc("POST /isalive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Starting API server on port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}

	log.Print("test")
}
