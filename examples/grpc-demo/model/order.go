package model

import "time"

// Order 订单模型
type Order struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	ProductID  int64     `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
