package webhook

type CheckResult struct {
	PaymentID string `json:"paymentId" validate:"required"`
	Passed    bool   `json:"passed"`
	Reason    string `json:"reason"`
}

type NetworkStatus struct {
	PaymentID string `json:"paymentId" validate:"required"`
	Status    string `json:"status" validate:"required|in:settled,returned"`
	Reason    string `json:"reason"`
}

func (r CheckResult) paymentID() string {
	return r.PaymentID
}

func (r NetworkStatus) paymentID() string {
	return r.PaymentID
}
