FROM golang:1.23-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY api/ .
RUN go build -o api-service .
CMD ["./api-service"]
