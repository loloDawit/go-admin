package product

import (
	"strconv"
	"time"
)

// CreateProductRequest is POST /api/v1/products' body.
type CreateProductRequest struct {
	SKU         string `json:"sku"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceMinor  int64  `json:"priceMinor"`
	Currency    string `json:"currency"`
}

// UpdateProductRequest is PATCH /api/v1/products/{id}'s body; every field is
// optional and a nil pointer leaves that column unchanged.
type UpdateProductRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	PriceMinor  *int64  `json:"priceMinor"`
	Currency    *string `json:"currency"`
}

// ProductResponse is the shape every route answers a single product with.
type ProductResponse struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PriceMinor  int64     `json:"priceMinor"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PageResponse is the response shape both List and Search answer with;
// pageSize is the effective size after clamping, not the caller's request.
type PageResponse struct {
	Items    []ProductResponse `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int               `json:"total"`
}

func newProductResponse(p Product) ProductResponse {
	return ProductResponse{
		ID:          strconv.FormatInt(p.ID, 10),
		SKU:         p.SKU,
		Title:       p.Title,
		Description: p.Description,
		PriceMinor:  p.PriceMinor,
		Currency:    p.Currency,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func newPageResponse(p Page) PageResponse {
	items := make([]ProductResponse, len(p.Items))
	for i, prod := range p.Items {
		items[i] = newProductResponse(prod)
	}
	return PageResponse{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total}
}
