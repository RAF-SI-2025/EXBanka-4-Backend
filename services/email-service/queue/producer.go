package queue

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "email.activation"
const ResetQueueName = "email.passwordreset"
const ConfirmQueueName = "email.passwordconfirmation"
const AccountCreatedQueueName = "email.accountcreated"
const CardConfirmationQueueName = "email.cardconfirmation"
const LoanLatePaymentQueueName = "email.loanlate"
const AccountLockedQueueName = "email.accountlocked"
const PaymentNotificationQueueName = "email.payment"
const CardBlockedQueueName = "email.cardblocked"
const LoanApprovedQueueName = "email.loanapproved"
const LimitChangeQueueName = "email.limitchange"

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

type CardConfirmationMessage struct {
	Email            string `json:"email"`
	FirstName        string `json:"first_name"`
	ConfirmationCode string `json:"confirmation_code"`
}

type LoanLatePaymentMessage struct {
	Email      string  `json:"email"`
	FirstName  string  `json:"first_name"`
	LoanNumber string  `json:"loan_number"`
	AmountDue  float64 `json:"amount_due"`
	Currency   string  `json:"currency"`
	RetryCount int32   `json:"retry_count"`
}

type AccountLockedMessage struct {
	Email             string `json:"email"`
	FirstName         string `json:"first_name"`
	PasswordResetLink string `json:"password_reset_link"`
}

type PaymentNotificationMessage struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	Direction    string  `json:"direction"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Counterparty string  `json:"counterparty"`
	AccountNumber string `json:"account_number"`
}

type CardBlockedMessage struct {
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	CardNumber string `json:"card_number"`
}

type LoanApprovedMessage struct {
	Email              string  `json:"email"`
	FirstName          string  `json:"first_name"`
	LoanAmount         float64 `json:"loan_amount"`
	Currency           string  `json:"currency"`
	MonthlyInstallment float64 `json:"monthly_installment"`
}

type LimitChangeMessage struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	DailyLimit   float64 `json:"daily_limit"`
	MonthlyLimit float64 `json:"monthly_limit"`
	Currency     string  `json:"currency"`
}

type Channel interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type Producer struct {
	ch Channel
}

func NewProducer(ch Channel) (*Producer, error) {
	queues := []string{QueueName, ResetQueueName, ConfirmQueueName, AccountCreatedQueueName,
		CardConfirmationQueueName, LoanLatePaymentQueueName, AccountLockedQueueName,
		PaymentNotificationQueueName, CardBlockedQueueName, LoanApprovedQueueName,
		LimitChangeQueueName}
	for _, q := range queues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return nil, err
		}
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

func (p *Producer) PublishCardConfirmation(msg CardConfirmationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", CardConfirmationQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishLoanLatePayment(msg LoanLatePaymentMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", LoanLatePaymentQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishAccountLocked(msg AccountLockedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", AccountLockedQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishPaymentNotification(msg PaymentNotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", PaymentNotificationQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishCardBlocked(msg CardBlockedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", CardBlockedQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishLoanApproved(msg LoanApprovedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", LoanApprovedQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishLimitChange(msg LimitChangeMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", LimitChangeQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
