package model

import "time"

// Product 商品数据模型
type Product struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	UpdatedAt time.Time `json:"updated_at"`
}
