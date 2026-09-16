package reporting

import (
	"strconv"
	"time"
)

type DashboardResponse struct {
	Counts  map[string]int        `json:"counts"`
	Recent  []RecentOrderResponse `json:"recent"`
	Revenue []RevenueDayResponse  `json:"revenue"`
}

type RecentOrderResponse struct {
	ID         string    `json:"id"`
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	TotalMinor int64     `json:"totalMinor"`
	Currency   string    `json:"currency"`
	PlacedAt   time.Time `json:"placedAt"`
}

type RevenueDayResponse struct {
	Day             string `json:"day"`
	Currency        string `json:"currency"`
	PlacedCount     int    `json:"placedCount"`
	PaidCount       int    `json:"paidCount"`
	RecognisedMinor int64  `json:"recognisedMinor"`
	RefundedMinor   int64  `json:"refundedMinor"`
	NetMinor        int64  `json:"netMinor"`
}

func newDashboardResponse(r Report) DashboardResponse {
	recent := make([]RecentOrderResponse, len(r.Recent))
	for i, o := range r.Recent {
		recent[i] = RecentOrderResponse{
			ID:         strconv.FormatInt(o.ID, 10),
			Number:     o.Number,
			Status:     o.Status,
			TotalMinor: o.TotalMinor,
			Currency:   o.Currency,
			PlacedAt:   o.PlacedAt,
		}
	}
	revenue := make([]RevenueDayResponse, len(r.Revenue))
	for i, d := range r.Revenue {
		revenue[i] = RevenueDayResponse{
			Day:             d.Day.Format(time.DateOnly),
			Currency:        d.Currency,
			PlacedCount:     d.PlacedCount,
			PaidCount:       d.PaidCount,
			RecognisedMinor: d.RecognisedMinor,
			RefundedMinor:   d.RefundedMinor,
			NetMinor:        d.NetMinor,
		}
	}
	return DashboardResponse{Counts: r.Counts, Recent: recent, Revenue: revenue}
}
