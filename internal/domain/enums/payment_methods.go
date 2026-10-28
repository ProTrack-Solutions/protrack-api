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
	// Venda paga inteira com vale-troca
	PaymentMethodStoreCredit PaymentMethod = "store_credit"
)

func (p PaymentMethod) IsValid() bool {
	switch p {
	case PaymentMethodCash, PaymentMethodCreditCard, PaymentMethodDebitCard, PaymentMethodPix,
		PaymentMethodBankTransfer, PaymentMethodInstallments, PaymentMethodOther, PaymentMethodStoreCredit:
		return true
	}
	return false
}
