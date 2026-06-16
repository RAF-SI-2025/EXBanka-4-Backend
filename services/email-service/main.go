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

	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/handlers"
	"github.com/RAF-SI-2025/EXBanka-4-Backend/services/email-service/queue"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/email"
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
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
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
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("rabbitmq conn close: %v", err)
		}
	}()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open publish channel: %v", err)
	}
	defer func() {
		if err := publishCh.Close(); err != nil {
			log.Printf("publish channel close: %v", err)
		}
	}()

	consumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open consume channel: %v", err)
	}
	defer func() {
		if err := consumeCh.Close(); err != nil {
			log.Printf("consume channel close: %v", err)
		}
	}()

	resetConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open reset consume channel: %v", err)
	}
	defer func() {
		if err := resetConsumeCh.Close(); err != nil {
			log.Printf("reset consume channel close: %v", err)
		}
	}()

	confirmConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open confirmation consume channel: %v", err)
	}
	defer func() {
		if err := confirmConsumeCh.Close(); err != nil {
			log.Printf("confirm consume channel close: %v", err)
		}
	}()

	accountCreatedConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open account created consume channel: %v", err)
	}
	defer func() {
		if err := accountCreatedConsumeCh.Close(); err != nil {
			log.Printf("account created consume channel close: %v", err)
		}
	}()

	cardConfirmConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open card confirmation consume channel: %v", err)
	}
	defer func() {
		if err := cardConfirmConsumeCh.Close(); err != nil {
			log.Printf("card confirm consume channel close: %v", err)
		}
	}()

	loanLateConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open loan late payment consume channel: %v", err)
	}
	defer func() {
		if err := loanLateConsumeCh.Close(); err != nil {
			log.Printf("loan late consume channel close: %v", err)
		}
	}()

	accountLockedConsumeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open account locked consume channel: %v", err)
	}
	defer func() {
		if err := accountLockedConsumeCh.Close(); err != nil {
			log.Printf("account locked consume channel close: %v", err)
		}
	}()

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

	cardConfirmTmpl, err := template.ParseFiles("templates/card_confirmation.html")
	if err != nil {
		log.Fatalf("failed to parse card confirmation email template: %v", err)
	}

	loanLateTmpl, err := template.ParseFiles("templates/loan_late_payment.html")
	if err != nil {
		log.Fatalf("failed to parse loan late payment email template: %v", err)
	}

	accountLockedTmpl, err := template.ParseFiles("templates/account_locked.html")
	if err != nil {
		log.Fatalf("failed to parse account locked email template: %v", err)
	}

	paymentNotifTmpl, err := template.ParseFiles("templates/payment_notification.html")
	if err != nil {
		log.Fatalf("failed to parse payment notification email template: %v", err)
	}
	cardBlockedTmpl, err := template.ParseFiles("templates/card_blocked.html")
	if err != nil {
		log.Fatalf("failed to parse card blocked email template: %v", err)
	}
	loanApprovedTmpl, err := template.ParseFiles("templates/loan_approved.html")
	if err != nil {
		log.Fatalf("failed to parse loan approved email template: %v", err)
	}
	limitChangeTmpl, err := template.ParseFiles("templates/limit_change.html")
	if err != nil {
		log.Fatalf("failed to parse limit change email template: %v", err)
	}

	orderStatusTmpl, err := template.ParseFiles("templates/order_status.html")
	if err != nil {
		log.Fatalf("failed to parse order status email template: %v", err)
	}

	priceAlertTmpl, err := template.ParseFiles("templates/price_alert.html")
	if err != nil {
		log.Fatalf("failed to parse price alert email template: %v", err)
	}

	paymentNotifCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open payment notification consume channel: %v", err)
	}
	defer func() { _ = paymentNotifCh.Close() }()
	cardBlockedCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open card blocked consume channel: %v", err)
	}
	defer func() { _ = cardBlockedCh.Close() }()
	loanApprovedCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open loan approved consume channel: %v", err)
	}
	defer func() { _ = loanApprovedCh.Close() }()
	limitChangeCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open limit change consume channel: %v", err)
	}
	defer func() { _ = limitChangeCh.Close() }()

	go queue.Consume(consumeCh, smtpCfg, tmpl)
	go queue.ConsumePasswordReset(resetConsumeCh, smtpCfg, resetTmpl)
	go queue.ConsumePasswordConfirmation(confirmConsumeCh, smtpCfg, confirmTmpl)
	go queue.ConsumeAccountCreated(accountCreatedConsumeCh, smtpCfg, accountCreatedTmpl)
	go queue.ConsumeCardConfirmation(cardConfirmConsumeCh, smtpCfg, cardConfirmTmpl)
	go queue.ConsumeLoanLatePayment(loanLateConsumeCh, smtpCfg, loanLateTmpl)
	go queue.ConsumeAccountLocked(accountLockedConsumeCh, smtpCfg, accountLockedTmpl)
	go queue.ConsumePaymentNotification(paymentNotifCh, smtpCfg, paymentNotifTmpl)
	go queue.ConsumeCardBlocked(cardBlockedCh, smtpCfg, cardBlockedTmpl)
	go queue.ConsumeLoanApproved(loanApprovedCh, smtpCfg, loanApprovedTmpl)
	go queue.ConsumeLimitChange(limitChangeCh, smtpCfg, limitChangeTmpl)

	orderStatusCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open order status consume channel: %v", err)
	}
	defer func() { _ = orderStatusCh.Close() }()

	priceAlertCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open price alert consume channel: %v", err)
	}
	defer func() { _ = priceAlertCh.Close() }()

	go queue.ConsumeOrderStatus(orderStatusCh, smtpCfg, orderStatusTmpl)
	go queue.ConsumePriceAlert(priceAlertCh, smtpCfg, priceAlertTmpl)

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
