package queue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"gopkg.in/gomail.v2"
)

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func Consume(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	msgs, err := ch.Consume(QueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	log.Println("email consumer started, waiting for messages")

	for d := range msgs {
		var msg ActivationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send activation email to %s: %v", msg.Email, err)
		} else {
			log.Printf("activation email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func sendEmail(cfg SMTPConfig, tmpl *template.Template, msg ActivationMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName":      msg.FirstName,
		"ActivationLink": msg.ActivationLink,
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Activate your AnkaBanka account")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func ConsumePasswordReset(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(ResetQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare password reset queue: %v", err)
	}

	msgs, err := ch.Consume(ResetQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start password reset consumer: %v", err)
	}

	log.Println("password reset email consumer started, waiting for messages")

	for d := range msgs {
		var msg PasswordResetMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode password reset message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendPasswordResetEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send password reset email to %s: %v", msg.Email, err)
		} else {
			log.Printf("password reset email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func sendPasswordResetEmail(cfg SMTPConfig, tmpl *template.Template, msg PasswordResetMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName": msg.FirstName,
		"ResetLink": msg.ResetLink,
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Reset your AnkaBanka password")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func ConsumeAccountCreated(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(AccountCreatedQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare account created queue: %v", err)
	}

	msgs, err := ch.Consume(AccountCreatedQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start account created consumer: %v", err)
	}

	log.Println("account created email consumer started, waiting for messages")

	for d := range msgs {
		var msg AccountCreatedMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode account created message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendAccountCreatedEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send account created email to %s: %v", msg.Email, err)
		} else {
			log.Printf("account created email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func sendAccountCreatedEmail(cfg SMTPConfig, tmpl *template.Template, msg AccountCreatedMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName":     msg.FirstName,
		"AccountName":   msg.AccountName,
		"AccountNumber": msg.AccountNumber,
		"CurrencyCode":  msg.CurrencyCode,
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Your AnkaBanka account has been created")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func ConsumeCardConfirmation(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(CardConfirmationQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare card confirmation queue: %v", err)
	}

	msgs, err := ch.Consume(CardConfirmationQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start card confirmation consumer: %v", err)
	}

	log.Println("card confirmation email consumer started, waiting for messages")

	for d := range msgs {
		var msg CardConfirmationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode card confirmation message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendCardConfirmationEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send card confirmation email to %s: %v", msg.Email, err)
		} else {
			log.Printf("card confirmation email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func sendCardConfirmationEmail(cfg SMTPConfig, tmpl *template.Template, msg CardConfirmationMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName":        msg.FirstName,
		"ConfirmationCode": msg.ConfirmationCode,
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Card Request Confirmation — AnkaBanka")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func ConsumePasswordConfirmation(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	msgs, err := ch.Consume(ConfirmQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start password confirmation consumer: %v", err)
	}

	log.Println("password confirmation email consumer started, waiting for messages")

	for d := range msgs {
		var msg PasswordConfirmationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode password confirmation message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendPasswordConfirmationEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send password confirmation email to %s: %v", msg.Email, err)
		} else {
			log.Printf("password confirmation email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func ConsumeLoanLatePayment(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(LoanLatePaymentQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare loan late payment queue: %v", err)
	}

	msgs, err := ch.Consume(LoanLatePaymentQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start loan late payment consumer: %v", err)
	}

	log.Println("loan late payment email consumer started, waiting for messages")

	for d := range msgs {
		var msg LoanLatePaymentMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode loan late payment message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendLoanLatePaymentEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send loan late payment email to %s: %v", msg.Email, err)
		} else {
			log.Printf("loan late payment email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func sendLoanLatePaymentEmail(cfg SMTPConfig, tmpl *template.Template, msg LoanLatePaymentMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName":  msg.FirstName,
		"LoanNumber": msg.LoanNumber,
		"AmountDue":  fmt.Sprintf("%.2f", msg.AmountDue),
		"Currency":   msg.Currency,
		"RetryCount": fmt.Sprintf("%d", msg.RetryCount),
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Loan Payment Failed — AnkaBanka")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func sendPasswordConfirmationEmail(cfg SMTPConfig, tmpl *template.Template, msg PasswordConfirmationMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName": msg.FirstName,
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Your AnkaBanka password has been set")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func ConsumeAccountLocked(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(AccountLockedQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare account locked queue: %v", err)
	}

	msgs, err := ch.Consume(AccountLockedQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start account locked consumer: %v", err)
	}

	log.Println("account locked email consumer started, waiting for messages")

	for d := range msgs {
		var msg AccountLockedMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode account locked message: %v", err)
			if err := d.Ack(false); err != nil {
				log.Printf("failed to ack message: %v", err)
			}
			continue
		}

		if err := sendAccountLockedEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send account locked email to %s: %v", msg.Email, err)
		} else {
			log.Printf("account locked email sent to %s", msg.Email)
		}

		if err := d.Ack(false); err != nil {
			log.Printf("failed to ack message: %v", err)
		}
	}
}

func sendAccountLockedEmail(cfg SMTPConfig, tmpl *template.Template, msg AccountLockedMessage) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"FirstName":         msg.FirstName,
		"PasswordResetLink": msg.PasswordResetLink,
	}); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", msg.Email)
	m.SetHeader("Subject", "Your AnkaBanka account has been locked")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}

func ConsumePaymentNotification(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(PaymentNotificationQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare payment notification queue: %v", err)
	}
	msgs, err := ch.Consume(PaymentNotificationQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start payment notification consumer: %v", err)
	}
	log.Println("payment notification email consumer started")
	for d := range msgs {
		var msg PaymentNotificationMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode payment notification message: %v", err)
			_ = d.Ack(false)
			continue
		}
		subject := "Payment Sent — AnkaBanka"
		if msg.Direction == "incoming" {
			subject = "Payment Received — AnkaBanka"
		}
		if err := sendTemplatedEmail(cfg, tmpl, "payment_notification.html", msg, msg.Email, subject); err != nil {
			log.Printf("failed to send payment notification email to %s: %v", msg.Email, err)
		}
		_ = d.Ack(false)
	}
}

func ConsumeCardBlocked(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(CardBlockedQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare card blocked queue: %v", err)
	}
	msgs, err := ch.Consume(CardBlockedQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start card blocked consumer: %v", err)
	}
	log.Println("card blocked email consumer started")
	for d := range msgs {
		var msg CardBlockedMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode card blocked message: %v", err)
			_ = d.Ack(false)
			continue
		}
		if err := sendTemplatedEmail(cfg, tmpl, "card_blocked.html", msg, msg.Email, "Your Card Has Been Blocked — AnkaBanka"); err != nil {
			log.Printf("failed to send card blocked email to %s: %v", msg.Email, err)
		}
		_ = d.Ack(false)
	}
}

func ConsumeLoanApproved(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(LoanApprovedQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare loan approved queue: %v", err)
	}
	msgs, err := ch.Consume(LoanApprovedQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start loan approved consumer: %v", err)
	}
	log.Println("loan approved email consumer started")
	for d := range msgs {
		var msg LoanApprovedMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode loan approved message: %v", err)
			_ = d.Ack(false)
			continue
		}
		if err := sendTemplatedEmail(cfg, tmpl, "loan_approved.html", msg, msg.Email, "Your Loan Has Been Approved — AnkaBanka"); err != nil {
			log.Printf("failed to send loan approved email to %s: %v", msg.Email, err)
		}
		_ = d.Ack(false)
	}
}

func ConsumeLimitChange(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(LimitChangeQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare limit change queue: %v", err)
	}
	msgs, err := ch.Consume(LimitChangeQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start limit change consumer: %v", err)
	}
	log.Println("limit change email consumer started")
	for d := range msgs {
		var msg LimitChangeMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode limit change message: %v", err)
			_ = d.Ack(false)
			continue
		}
		if err := sendTemplatedEmail(cfg, tmpl, "limit_change.html", msg, msg.Email, "Account Limit Updated — AnkaBanka"); err != nil {
			log.Printf("failed to send limit change email to %s: %v", msg.Email, err)
		}
		_ = d.Ack(false)
	}
}

func ConsumeOrderStatus(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(OrderStatusQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare order status queue: %v", err)
	}
	msgs, err := ch.Consume(OrderStatusQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start order status consumer: %v", err)
	}
	log.Println("order status email consumer started")
	for d := range msgs {
		var msg OrderStatusMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode order status message: %v", err)
			_ = d.Ack(false)
			continue
		}
		subject := fmt.Sprintf("Order %s — AnkaBanka", msg.Status)
		if err := sendTemplatedEmail(cfg, tmpl, "order_status.html", msg, msg.Email, subject); err != nil {
			log.Printf("failed to send order status email to %s: %v", msg.Email, err)
		}
		_ = d.Ack(false)
	}
}

func ConsumePriceAlert(ch *amqp.Channel, cfg SMTPConfig, tmpl *template.Template) {
	if _, err := ch.QueueDeclare(PriceAlertQueueName, true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare price alert queue: %v", err)
	}
	msgs, err := ch.Consume(PriceAlertQueueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start price alert consumer: %v", err)
	}
	log.Println("price alert email consumer started")
	for d := range msgs {
		var msg PriceAlertMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("failed to decode price alert message: %v", err)
			_ = d.Ack(false)
			continue
		}
		if err := sendTemplatedEmail(cfg, tmpl, "price_alert.html", msg, msg.Email, "Price Alert Triggered — AnkaBanka"); err != nil {
			log.Printf("failed to send price alert email to %s: %v", msg.Email, err)
		}
		_ = d.Ack(false)
	}
}

func sendTemplatedEmail(cfg SMTPConfig, tmpl *template.Template, _ string, data interface{}, to, subject string) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", buf.String())
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}
