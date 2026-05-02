package domain

type Payment struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	// CustomerEmail is not persisted; it travels with the payment event only.
	CustomerEmail string `json:"customer_email,omitempty"`
}
