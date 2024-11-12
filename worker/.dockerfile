FROM golang:1.20-alpine
WORKDIR /app
COPY ./worker .
RUN go mod tidy && go build -o transaction-worker
CMD ["/app/transaction-worker"]