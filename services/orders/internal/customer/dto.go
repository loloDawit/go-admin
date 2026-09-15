package customer

import (
	"strconv"
	"time"
)

// CreateCustomerRequest is POST /api/v1/customers' body.
type CreateCustomerRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CustomerResponse is the shape every route answers a single customer with.
type CustomerResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PageResponse is the response shape List answers with; pageSize is the
// effective size after clamping, not the caller's request.
type PageResponse struct {
	Items    []CustomerResponse `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Total    int                `json:"total"`
}

// LifetimeValueResponse reports minor-unit revenue, currency-naive: correct
// for a single-currency shop, not summed correctly across currencies.
type LifetimeValueResponse struct {
	LifetimeValueMinor int64 `json:"lifetimeValueMinor"`
}

func newCustomerResponse(c Customer) CustomerResponse {
	return CustomerResponse{
		ID:        strconv.FormatInt(c.ID, 10),
		Email:     c.Email,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func newPageResponse(p Page) PageResponse {
	items := make([]CustomerResponse, len(p.Items))
	for i, c := range p.Items {
		items[i] = newCustomerResponse(c)
	}
	return PageResponse{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total}
}
