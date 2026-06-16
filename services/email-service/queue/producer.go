package queue

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "email.activation"
const OrderStatusQueueName = "email.orderstatus"
const PriceAlertQueueName = "email.pricealert"
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
const OtcCounterOfferQueueName = "email.otc.counteroffer"
const OtcStatusChangeQueueName = "email.otc.statuschange"
const OtcContractExpiryQueueName = "email.otc.expiry"

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
	Email         string  `json:"email"`
	FirstName     string  `json:"first_name"`
	Direction     string  `json:"direction"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Counterparty  string  `json:"counterparty"`
	AccountNumber string  `json:"account_number"`
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

type OrderStatusMessage struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	OrderID      int64   `json:"order_id"`
	Ticker       string  `json:"ticker"`
	Direction    string  `json:"direction"`
	Status       string  `json:"status"`
	Quantity     int32   `json:"quantity"`
	FilledQty    int32   `json:"filled_qty"`
	PricePerUnit float64 `json:"price_per_unit"`
}

type PriceAlertMessage struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	Ticker       string  `json:"ticker"`
	Condition    string  `json:"condition"`
	Threshold    float64 `json:"threshold"`
	CurrentPrice float64 `json:"current_price"`
}

type LimitChangeMessage struct {
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	DailyLimit   float64 `json:"daily_limit"`
	MonthlyLimit float64 `json:"monthly_limit"`
	Currency     string  `json:"currency"`
}

type OtcCounterOfferMessage struct {
	Email             string  `json:"email"`
	FirstName         string  `json:"first_name"`
	NegotiationId     int64   `json:"negotiation_id"`
	Ticker            string  `json:"ticker"`
	NewAmount         int32   `json:"new_amount"`
	NewPricePerStock  float64 `json:"new_price_per_stock"`
	NewPremium        float64 `json:"new_premium"`
	NewSettlementDate string  `json:"new_settlement_date"`
	CounterPartyName  string  `json:"counter_party_name"`
}

type OtcStatusChangeMessage struct {
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	NegotiationId int64  `json:"negotiation_id"`
	Ticker        string `json:"ticker"`
	Status        string `json:"status"`
}

type OtcContractExpiryMessage struct {
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	ContractId int64  `json:"contract_id"`
	Ticker     string `json:"ticker"`
	ExpiryDate string `json:"expiry_date"`
	DaysLeft   int    `json:"days_left"`
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
		LimitChangeQueueName, OrderStatusQueueName, PriceAlertQueueName,
		OtcCounterOfferQueueName, OtcStatusChangeQueueName, OtcContractExpiryQueueName}
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

func (p *Producer) PublishOrderStatus(msg OrderStatusMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", OrderStatusQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Producer) PublishPriceAlert(msg PriceAlertMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.ch.Publish("", PriceAlertQueueName, false, false, amqp.Publishing{
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
