package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	apiURL = "http://localhost:8080/transfer"
)

type TransactionRequest struct {
	FromAccountID int     `json:"from_account_id"`
	ToAccountID   int     `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

func TestTransferAPI(t *testing.T) {
	time.Sleep(5 * time.Second)

	reqBody := TransactionRequest{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        100.0,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestMain(m *testing.M) {
	err := runDockerCompose()
	if err != nil {
		os.Exit(1)
	}

	code := m.Run()

	err = stopDockerCompose()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(code)
}

func runDockerCompose() error {
	cmd := exec.Command("docker-compose", "up", "--build", "-d")
	return cmd.Run()
}

func stopDockerCompose() error {
	cmd := exec.Command("docker-compose", "down")
	return cmd.Run()
}
