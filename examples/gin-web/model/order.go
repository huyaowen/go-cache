package model

import "time"

// Order 订单模型
type Order struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
