package enums

type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodCreditCard   PaymentMethod = "credit_card"
	PaymentMethodDebitCard    PaymentMethod = "debit_card"
	PaymentMethodPix          PaymentMethod = "pix"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodInstallments PaymentMethod = "installments"
	PaymentMethodOther        PaymentMethod = "other"
)
