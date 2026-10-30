package payment

type RequestPayment struct {
	AccountID    string `json:"accountId" validate:"required"`
	CreditorIBAN string `json:"creditorIban" validate:"required"`
	Amount       int64  `json:"amount" validate:"required|min:1"`
}
