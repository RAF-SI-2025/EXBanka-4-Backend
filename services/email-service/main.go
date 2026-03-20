package main

import (
	"html/template"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/handlers"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/queue"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func main() {
	var err error
	smtpPort, err := strconv.Atoi(mustEnv("SMTP_PORT"))
	if err != nil {
		log.Fatalf("SMTP_PORT must be an integer: %v", err)
	}

	smtpCfg := queue.SMTPConfig{
		Host:     mustEnv("SMTP_HOST"),
		Port:     smtpPort,
		User:     mustEnv("SMTP_USER"),
		Password: mustEnv("SMTP_PASSWORD"),
		From:     mustEnv("FROM_EMAIL"),
	}

	rabbitmqURL := mustEnv("RABBITMQ_URL")
	var conn *amqp.Connection
	for i := 1; i <= 20; i++ {
		conn, err = amqp.Dial(rabbitmqURL)
		if err == nil {
			break
		}
		log.Printf("RabbitMQ not ready (attempt %d/20): %v", i, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ after 20 attempts: %v", err)
	}
	defer conn.Close()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open publish channel: %v", err)
	}
	defer publishCh.Close()

	consumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open consume channel: %v", err)
	}
	defer consumeCh.Close()

	resetConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open reset consume channel: %v", err)
	}
	defer resetConsumeCh.Close()

	confirmConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open confirmation consume channel: %v", err)
	}
	defer confirmConsumeCh.Close()

	accountCreatedConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open account created consume channel: %v", err)
	}
	defer accountCreatedConsumeCh.Close()

	producer, err := queue.NewProducer(publishCh)
	if err != nil {
		log.Fatalf("failed to create producer: %v", err)
	}

	tmpl, err := template.ParseFiles("templates/activation.html")
	if err != nil {
		log.Fatalf("failed to parse email template: %v", err)
	}

	resetTmpl, err := template.ParseFiles("templates/password_reset.html")
	if err != nil {
		log.Fatalf("failed to parse password reset email template: %v", err)
	}

	confirmTmpl, err := template.ParseFiles("templates/password_confirmation.html")
	if err != nil {
		log.Fatalf("failed to parse password confirmation email template: %v", err)
	}

	accountCreatedTmpl, err := template.ParseFiles("templates/account_created.html")
	if err != nil {
		log.Fatalf("failed to parse account created email template: %v", err)
	}

	go queue.Consume(consumeCh, smtpCfg, tmpl)
	go queue.ConsumePasswordReset(resetConsumeCh, smtpCfg, resetTmpl)
	go queue.ConsumePasswordConfirmation(confirmConsumeCh, smtpCfg, confirmTmpl)
	go queue.ConsumeAccountCreated(accountCreatedConsumeCh, smtpCfg, accountCreatedTmpl)

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterEmailServiceServer(s, &handlers.EmailServer{Producer: producer})

	log.Println("email-service listening on :50053")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
