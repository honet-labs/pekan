package domain

import "strings"

const (
	MaxDescriptionLength   = 1000
	MaxMerchantNameLength  = 255
	MaxReceiptNumberLength = 100
	MaxPaymentMethodLength = 100
	MaxItemNameLength      = 255
	MaxNotesLength         = 1000
)

func ValidateCreateInput(accountID string, amountMinor int64, currency string, trxType TransactionType) error {
	if strings.TrimSpace(accountID) == "" {
		return ErrInvalidAccount
	}
	if amountMinor <= 0 {
		return ErrInvalidAmount
	}
	if len(strings.TrimSpace(currency)) != 3 {
		return ErrInvalidCurrency
	}
	if trxType != TransactionTypeIncome &&
		trxType != TransactionTypeExpense &&
		trxType != TransactionTypeTransfer &&
		trxType != TransactionTypeSavings {
		return ErrInvalidType
	}
	return nil
}

func ValidateStringLengths(description, merchant, receipt, payment *string) error {
	if description != nil && len(*description) > MaxDescriptionLength {
		return ErrInputTooLong
	}
	if merchant != nil && len(*merchant) > MaxMerchantNameLength {
		return ErrInputTooLong
	}
	if receipt != nil && len(*receipt) > MaxReceiptNumberLength {
		return ErrInputTooLong
	}
	if payment != nil && len(*payment) > MaxPaymentMethodLength {
		return ErrInputTooLong
	}
	return nil
}
