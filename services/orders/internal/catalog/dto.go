package catalog

// Product is the purchase-time snapshot Catalog resolves an id to. An id with
// no matching product is simply absent from Resolve's result, not an error.
type Product struct {
	ID         string
	Title      string
	PriceMinor int64
	Currency   string
	Status     string
}

type resolveRequest struct {
	IDs []string `json:"ids"`
}

type resolveResponse struct {
	Products []productDTO `json:"products"`
}

type productDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceMinor int64  `json:"priceMinor"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
}

func (r resolveResponse) toProducts() []Product {
	products := make([]Product, len(r.Products))
	for i, p := range r.Products {
		products[i] = Product{
			ID:         p.ID,
			Title:      p.Title,
			PriceMinor: p.PriceMinor,
			Currency:   p.Currency,
			Status:     p.Status,
		}
	}
	return products
}
