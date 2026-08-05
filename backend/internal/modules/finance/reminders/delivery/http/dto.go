package http

type ReminderRequest struct {
	Title          string  `json:"title"`
	Description    *string `json:"description"`
	AmountMinor    *int64  `json:"amount_minor"`
	Currency       *string `json:"currency"`
	DueDate        string  `json:"due_date"`
	RepeatInterval string  `json:"repeat_interval"`
	Status         string  `json:"status"`
	TotalTenor     *int    `json:"total_tenor"`
	CurrentTenor   *int    `json:"current_tenor"`
}

type ReminderStatusRequest struct {
	Status string `json:"status"`
}

type ReminderResponse struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    *string `json:"description"`
	AmountMinor    *int64  `json:"amount_minor"`
	Currency       *string `json:"currency"`
	DueDate        string  `json:"due_date"`
	RepeatInterval string  `json:"repeat_interval"`
	Status         string  `json:"status"`
	TotalTenor     *int    `json:"total_tenor"`
	CurrentTenor   *int    `json:"current_tenor"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type ReminderPaymentRequest struct {
	PaidAt        string  `json:"paid_at"`
	AmountMinor   int64   `json:"amount_minor"`
	Status        string  `json:"status"`
	Notes         *string `json:"notes"`
	ProofImageURL *string `json:"proof_image_url"`
}

type ReminderPaymentResponse struct {
	ID            string  `json:"id"`
	ReminderID    string  `json:"reminder_id"`
	PaidAt        string  `json:"paid_at"`
	AmountMinor   int64   `json:"amount_minor"`
	Status        string  `json:"status"`
	Notes         *string `json:"notes"`
	ProofImageURL *string `json:"proof_image_url"`
	CreatedAt     string  `json:"created_at"`
}

