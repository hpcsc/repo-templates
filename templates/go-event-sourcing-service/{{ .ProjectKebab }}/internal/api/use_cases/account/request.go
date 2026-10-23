package account

type OpenRequest struct {
	Owner string `json:"owner" validate:"required"`
}

type AmountRequest struct {
	Amount int64 `json:"amount" validate:"required|min:1"`
}
