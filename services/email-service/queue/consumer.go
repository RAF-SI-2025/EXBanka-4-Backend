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
	m.SetHeader("Subject", "Activate your EXBanka account")
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)
	return d.DialAndSend(m)
}
