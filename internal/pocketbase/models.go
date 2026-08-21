package pocketbase

type Page[T any] struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
	Items      []T `json:"items"`
}

type Supermarket struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	StatsSchedule string `json:"stats_schedule"`
}

type Product struct {
	ID            string `json:"id"`
	SupermarketID string `json:"supermarket_id"`
}

type Price struct {
	ProductID string   `json:"product_id"`
	Date      string   `json:"date"`
	BulkPrice *float64 `json:"bulk_price"`
}

type ProductStats struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	LastUpdated string `json:"last_updated"`
}

type Job struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}
