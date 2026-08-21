package pocketbase

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Repository struct{ client *Client }

func NewRepository(client *Client) *Repository { return &Repository{client: client} }

func (r *Repository) Supermarkets(ctx context.Context) ([]Supermarket, error) {
	return listAll[Supermarket](ctx, r.client, "supermarkets", "enabled = true")
}

func (r *Repository) Products(ctx context.Context, supermarketID string) ([]Product, error) {
	filter := fmt.Sprintf("supermarket_id = '%s'", escapeFilter(supermarketID))
	products, err := listAll[Product](ctx, r.client, "products", filter)
	if err != nil { return nil, err }
	for _, product := range products {
		if product.SupermarketID != supermarketID { return nil, fmt.Errorf("product %s belongs to another supermarket", product.ID) }
	}
	return products, nil
}

func (r *Repository) Supermarket(ctx context.Context, id string) (*Supermarket, error) {
	var supermarket Supermarket
	if err := r.client.Get(ctx, "supermarkets", id, &supermarket); err != nil { return nil, err }
	return &supermarket, nil
}

func (r *Repository) Prices(ctx context.Context, productID string) ([]Price, error) {
	filter := fmt.Sprintf("product_id = '%s'", escapeFilter(productID))
	prices, err := listAll[Price](ctx, r.client, "prices", filter)
	if err != nil { return nil, err }
	// PocketBase's filter selects the records; sorting here keeps the contract explicit.
	sort.Slice(prices, func(i, j int) bool { return prices[i].Date < prices[j].Date })
	return prices, nil
}

func (r *Repository) FindStats(ctx context.Context, productID string) (*ProductStats, error) {
	var page Page[ProductStats]
	filter := fmt.Sprintf("product_id = '%s'", escapeFilter(productID))
	if err := r.client.List(ctx, "product_stats", values(1, 2, filter), &page); err != nil { return nil, err }
	if len(page.Items) == 0 { return nil, nil }
	return &page.Items[0], nil
}

func (r *Repository) CreateStats(ctx context.Context, data any) error {
	return r.client.Create(ctx, "product_stats", data, nil)
}

func (r *Repository) UpdateStats(ctx context.Context, id string, data any) error {
	return r.client.Update(ctx, "product_stats", id, data, nil)
}

func (r *Repository) CreateJob(ctx context.Context, data any, out *Job) error {
	return r.client.Create(ctx, "jobs", data, out)
}

func (r *Repository) UpdateJob(ctx context.Context, id string, data any) error {
	return r.client.Update(ctx, "jobs", id, data, nil)
}

func values(page, perPage int, filter string) url.Values {
	v := url.Values{}
	v.Set("page", fmt.Sprint(page))
	v.Set("perPage", fmt.Sprint(perPage))
	if filter != "" { v.Set("filter", filter) }
	return v
}

func escapeFilter(value string) string { return strings.ReplaceAll(value, "'", "\\'") }

func listAll[T any](ctx context.Context, client *Client, collection, filter string) ([]T, error) {
	var all []T
	for pageNumber := 1; ; pageNumber++ {
		var page Page[T]
		if err := client.List(ctx, collection, values(pageNumber, 500, filter), &page); err != nil { return nil, err }
		all = append(all, page.Items...)
		if pageNumber >= page.TotalPages || len(page.Items) == 0 { return all, nil }
	}
}
