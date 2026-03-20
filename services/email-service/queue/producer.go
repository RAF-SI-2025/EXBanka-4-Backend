package queue

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName              = "email.activation"
const ResetQueueName         = "email.passwordreset"
const ConfirmQueueName       = "email.passwordconfirmation"
const AccountCreatedQueueName = "email.accountcreated"

type ActivationMessage struct {
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	ActivationLink string `json:"activation_link"`
}

type PasswordResetMessage struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	ResetLink string `json:"reset_link"`
}

type PasswordConfirmationMessage struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}

type AccountCreatedMessage struct {
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	CurrencyCode  string `json:"currency_code"`
}

type Producer struct {
	ch *amqp.Channel
}

func NewProducer(ch *amqp.Channel) (*Producer, error) {
	if _, err := ch.QueueDeclare(QueueName, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(ResetQueueName, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(ConfirmQueueName, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(AccountCreatedQueueName, true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &Producer{ch: ch}, nil
}

func (p *Producer) Publish(msg ActivationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", QueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishPasswordReset(msg PasswordResetMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", ResetQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishPasswordConfirmation(msg PasswordConfirmationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", ConfirmQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishAccountCreated(msg AccountCreatedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", AccountCreatedQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
