package transfer

type StartRequest struct {
	From   string `json:"from" validate:"required"`
	To     string `json:"to" validate:"required"`
	Amount int64  `json:"amount" validate:"required|min:1"`
}
