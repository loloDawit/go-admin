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

// ResolveRequest is POST /internal/products/resolve's body; ids are strings
// on the wire, parsed the same as a path id.
type ResolveRequest struct {
	IDs []string `json:"ids"`
}

// ResolvedProduct is the snapshot shape Orders needs at purchase time, not
// the full ProductResponse.
type ResolvedProduct struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceMinor int64  `json:"priceMinor"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
}

// ResolveResponse: products are ordered to match the order of the requested
// ids; an id with no matching product is simply absent, never an error.
type ResolveResponse struct {
	Products []ResolvedProduct `json:"products"`
}

func newResolvedProduct(p Product) ResolvedProduct {
	return ResolvedProduct{
		ID:         strconv.FormatInt(p.ID, 10),
		Title:      p.Title,
		PriceMinor: p.PriceMinor,
		Currency:   p.Currency,
		Status:     string(p.Status),
	}
}

func newResolveResponse(items []Product) ResolveResponse {
	products := make([]ResolvedProduct, len(items))
	for i, p := range items {
		products[i] = newResolvedProduct(p)
	}
	return ResolveResponse{Products: products}
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
