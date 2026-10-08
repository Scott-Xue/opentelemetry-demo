package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// Decisions returned to callers.
const (
	DecisionApprove = "APPROVE"
	DecisionReview  = "REVIEW"
	DecisionReject  = "REJECT"
)

type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type Address struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       string `json:"zip_code"`
}

type ScreenRequest struct {
	OrderID         string  `json:"order_id"`
	UserID          string  `json:"user_id"`
	Email           string  `json:"email"`
	CardBIN         string  `json:"card_bin"`
	CardLast4       string  `json:"card_last4"`
	Amount          Money   `json:"amount"`
	ShippingAddress Address `json:"shipping_address"`
	Items           []Item  `json:"items"`
}

type ScreenResponse struct {
	Decision    string   `json:"decision"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons"`
	ProviderRef string   `json:"provider_ref"`
}

// Screen scores one order. It is a pure function: the same request always yields the
// same response, so tests are stable and two experiment runs over the same orders are
// comparable. A real vendor would not be deterministic, but a non-deterministic stand-in
// would make it impossible to tell an experiment effect from vendor noise.
func Screen(req ScreenRequest) ScreenResponse {
	sum := sha256.Sum256([]byte(req.OrderID))

	// Baseline "model" score in [0,39], standing in for the vendor's own risk model.
	score := int(binary.BigEndian.Uint32(sum[0:4]) % 40)
	reasons := []string{}

	if req.Amount.Units >= 1000 {
		score += 25
		reasons = append(reasons, "high_value")
	}
	for _, it := range req.Items {
		if it.Quantity >= 10 {
			score += 20
			reasons = append(reasons, "quantity_anomaly")
			break
		}
	}
	if req.ShippingAddress.ZipCode == "" || req.ShippingAddress.Country == "" ||
		req.ShippingAddress.StreetAddress == "" {
		score += 15
		reasons = append(reasons, "incomplete_address")
	}
	if req.ShippingAddress.Country != "" && req.ShippingAddress.Country != "US" {
		score += 10
		reasons = append(reasons, "cross_border")
	}
	if len(req.CardBIN) != 6 || len(req.CardLast4) != 4 {
		score += 15
		reasons = append(reasons, "missing_card_signals")
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	decision := DecisionApprove
	switch {
	case score >= 70:
		decision = DecisionReject
	case score >= 40:
		decision = DecisionReview
	}

	return ScreenResponse{
		Decision:    decision,
		Score:       score,
		Reasons:     reasons,
		ProviderRef: "fd-" + hex.EncodeToString(sum[:])[:12],
	}
}
