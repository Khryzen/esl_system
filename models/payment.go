package models

import (
	"errors"
	"strings"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

const (
	PaymentMethodCash         = "Cash"
	PaymentMethodGCash        = "GCash"
	PaymentMethodBankTransfer = "Bank Transfer"
	PaymentMethodAlipay       = "Alipay"
	PaymentMethodWeChatPay    = "WeChat Pay"
	PaymentMethodMaya         = "Maya"
	PaymentMethodOthers       = "Others"
)

var (
	ErrPaymentInvoiceRequired = errors.New(
		"Invoice is required.",
	)
	ErrPaymentAmountRequired = errors.New(
		"Payment amount must be greater than zero.",
	)
	ErrPaymentMethodRequired = errors.New(
		"Payment method is required.",
	)
	ErrPaymentMethodInvalid = errors.New(
		"The selected payment method is invalid.",
	)
)

type Payment struct {
	uadmin.Model
	Invoice         Invoice
	InvoiceID       uint
	Amount          float64 `uadmin:"required"`
	PaymentDate     time.Time
	PaymentMethod   string `uadmin:"required"`
	ReferenceNumber string
	Notes           string
}

func (p *Payment) Create() error {
	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		return p.CreateWithTx(tx)
	})
}

func (p *Payment) CreateWithTx(tx *gorm.DB) error {
	if p.InvoiceID == 0 {
		return ErrPaymentInvoiceRequired
	}

	if p.Amount <= 0 {
		return ErrPaymentAmountRequired
	}

	p.PaymentMethod = strings.TrimSpace(p.PaymentMethod)

	if !isValidPaymentMethod(p.PaymentMethod) {
		return ErrPaymentMethodInvalid
	}

	var invoice Invoice

	if err := tx.First(&invoice, p.InvoiceID).Error; err != nil {
		return err
	}

	p.Invoice = invoice
	paymentDate := time.Now().In(time.Local)
	p.PaymentDate = paymentDate
	return tx.Create(p).Error
}

func isValidPaymentMethod(method string) bool {
	switch method {
	case PaymentMethodCash,
		PaymentMethodGCash,
		PaymentMethodBankTransfer,
		PaymentMethodAlipay,
		PaymentMethodWeChatPay,
		PaymentMethodMaya,
		PaymentMethodOthers:
		return true
	default:
		return false
	}
}

func PaymentMethods() []string {
	return []string{
		PaymentMethodCash,
		PaymentMethodGCash,
		PaymentMethodBankTransfer,
		PaymentMethodAlipay,
		PaymentMethodWeChatPay,
		PaymentMethodMaya,
		PaymentMethodOthers,
	}
}
