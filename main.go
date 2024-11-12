package main

import (
	"log"
	"net/http"
	"os"

	dotenv "github.com/joho/godotenv"
	ampq "github.com/rabbitmq/amqp091-go"
)

type RequestBody struct {
	FromAccountId int `json:"from_account"`
	ToAccountId   int `json:"to_account"`

	Amount float32 `json:"amount"`
}

func transferMoney(w http.ResponseWriter, r *http.Request) {

}

func main() {
	err := dotenv.Load()
	if err != nil {
		log.Fatalf("Unable to load env file. Error: %s", err)
	}

	conn, err := ampq.Dial(
		"amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@" + os.Getenv("RABBITMQ_HOST") + ":" + os.Getenv("RABBITMQ_PORT") + "/",
	)

	if err != nil {
		log.Fatalf("Unable to establish connection. Error: %s", err)
	}

	defer func() {
		if conn.Close() != nil {
			log.Fatalf("Unable to close connection. Error: %s", err)
		}
	}()

}
