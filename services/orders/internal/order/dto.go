package order

import (
	"strconv"
	"time"
)

// CreateOrderRequest is POST /api/v1/orders' body; ids are strings on the
// wire, parsed to their internal integer form at the handler boundary.
type CreateOrderRequest struct {
	CustomerID string                   `json:"customerId"`
	Items      []CreateOrderItemRequest `json:"items"`
}

type CreateOrderItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// OrderResponse is the shape every route answers an order with.
type OrderResponse struct {
	ID         string              `json:"id"`
	Number     string              `json:"number"`
	CustomerID string              `json:"customerId"`
	Status     string              `json:"status"`
	TotalMinor int64               `json:"totalMinor"`
	Currency   string              `json:"currency"`
	PlacedAt   time.Time           `json:"placedAt"`
	Items      []OrderItemResponse `json:"items"`
}

// SetStatusRequest is POST /orders/{id}/status' body; it carries no actor
// field; the actor comes from the verified principal, never the request.
type SetStatusRequest struct {
	Status string `json:"status"`
}

// TransitionRequest is Cancel and Refund's shared body shape.
type TransitionRequest struct {
	Reason string `json:"reason"`
}

// OrderListItemResponse is a listing row: header fields only, no items,
// since fetching every line for every row in a page is not what a list is for.
type OrderListItemResponse struct {
	ID         string    `json:"id"`
	Number     string    `json:"number"`
	CustomerID string    `json:"customerId"`
	Status     string    `json:"status"`
	TotalMinor int64     `json:"totalMinor"`
	Currency   string    `json:"currency"`
	PlacedAt   time.Time `json:"placedAt"`
}

// OrderPageResponse's PageSize is the effective size after clamping, not the
// caller's request.
type OrderPageResponse struct {
	Items    []OrderListItemResponse `json:"items"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
	Total    int                     `json:"total"`
}

func newOrderPageResponse(p Page) OrderPageResponse {
	items := make([]OrderListItemResponse, len(p.Items))
	for i, o := range p.Items {
		items[i] = OrderListItemResponse{
			ID:         strconv.FormatInt(o.ID, 10),
			Number:     o.Number,
			CustomerID: strconv.FormatInt(o.CustomerID, 10),
			Status:     string(o.Status),
			TotalMinor: o.TotalMinor,
			Currency:   o.Currency,
			PlacedAt:   o.PlacedAt,
		}
	}
	return OrderPageResponse{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total}
}

type OrderItemResponse struct {
	ProductID      string `json:"productId"`
	TitleSnapshot  string `json:"titleSnapshot"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
	Currency       string `json:"currency"`
	Quantity       int    `json:"quantity"`
	LineTotalMinor int64  `json:"lineTotalMinor"`
}

func newOrderResponse(o Order) OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, it := range o.Items {
		items[i] = OrderItemResponse{
			ProductID:      strconv.FormatInt(it.ProductID, 10),
			TitleSnapshot:  it.TitleSnapshot,
			UnitPriceMinor: it.UnitPriceMinor,
			Currency:       it.Currency,
			Quantity:       it.Quantity,
			LineTotalMinor: it.LineTotalMinor,
		}
	}
	return OrderResponse{
		ID:         strconv.FormatInt(o.ID, 10),
		Number:     o.Number,
		CustomerID: strconv.FormatInt(o.CustomerID, 10),
		Status:     string(o.Status),
		TotalMinor: o.TotalMinor,
		Currency:   o.Currency,
		PlacedAt:   o.PlacedAt,
		Items:      items,
	}
}
