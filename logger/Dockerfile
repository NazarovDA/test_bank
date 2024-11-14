FROM golang:1.23-alpine
WORKDIR /app
COPY go.mod ./
RUN go mod tidy
COPY logger/ .
RUN go build -o transaction-logger .
CMD ["./transaction-logger"]
