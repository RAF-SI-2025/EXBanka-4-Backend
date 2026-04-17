FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /service ./services/portfolio-service

FROM alpine:3.20
COPY --from=builder /service /service
ENTRYPOINT ["/service"]
