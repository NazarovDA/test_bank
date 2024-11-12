FROM golang:1.20-alpine
WORKDIR /app
COPY ./api .
RUN go mod tidy && go build -o api-service
CMD ["/app/api-service"]
