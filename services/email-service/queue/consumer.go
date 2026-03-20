package queue

import (
	"bytes"
	"encoding/json"
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
			d.Ack(false)
			continue
		}

		if err := sendEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send activation email to %s: %v", msg.Email, err)
		} else {
			log.Printf("activation email sent to %s", msg.Email)
		}

		d.Ack(false)
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
			d.Ack(false)
			continue
		}

		if err := sendPasswordResetEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send password reset email to %s: %v", msg.Email, err)
		} else {
			log.Printf("password reset email sent to %s", msg.Email)
		}

		d.Ack(false)
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
			d.Ack(false)
			continue
		}

		if err := sendAccountCreatedEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send account created email to %s: %v", msg.Email, err)
		} else {
			log.Printf("account created email sent to %s", msg.Email)
		}

		d.Ack(false)
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
			d.Ack(false)
			continue
		}

		if err := sendPasswordConfirmationEmail(cfg, tmpl, msg); err != nil {
			log.Printf("failed to send password confirmation email to %s: %v", msg.Email, err)
		} else {
			log.Printf("password confirmation email sent to %s", msg.Email)
		}

		d.Ack(false)
	}
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
