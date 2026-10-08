package main

import "testing"

func baseRequest() ScreenRequest {
	return ScreenRequest{
		OrderID:   "2b1e9f00-0000-4000-8000-000000000001",
		UserID:    "user-1",
		Email:     "someone@example.com",
		CardBIN:   "443280",
		CardLast4: "0454",
		Amount:    Money{CurrencyCode: "USD", Units: 100, Nanos: 0},
		ShippingAddress: Address{
			StreetAddress: "1600 Amphitheatre Parkway",
			City:          "Mountain View",
			State:         "CA",
			Country:       "US",
			ZipCode:       "94043",
		},
		Items: []Item{{ProductID: "OLJCESPC7Z", Quantity: 1}},
	}
}

func TestScreenIsDeterministic(t *testing.T) {
	a := Screen(baseRequest())
	b := Screen(baseRequest())
	if a.Score != b.Score || a.Decision != b.Decision || a.ProviderRef != b.ProviderRef {
		t.Fatalf("not deterministic: %+v vs %+v", a, b)
	}
}

func TestProviderRefDerivedFromOrderID(t *testing.T) {
	r := Screen(baseRequest())
	if len(r.ProviderRef) != 15 || r.ProviderRef[:3] != "fd-" {
		t.Fatalf("unexpected provider_ref %q", r.ProviderRef)
	}
	other := baseRequest()
	other.OrderID = "different-order"
	if Screen(other).ProviderRef == r.ProviderRef {
		t.Fatal("provider_ref must differ for a different order id")
	}
}

func TestHighValueRaisesScore(t *testing.T) {
	low := Screen(baseRequest())
	req := baseRequest()
	req.Amount.Units = 5000
	high := Screen(req)
	if high.Score <= low.Score {
		t.Fatalf("expected high-value order to score higher: %d vs %d", high.Score, low.Score)
	}
	if !hasReason(high.Reasons, "high_value") {
		t.Fatalf("expected high_value reason, got %v", high.Reasons)
	}
}

func TestQuantityAnomaly(t *testing.T) {
	req := baseRequest()
	req.Items = []Item{{ProductID: "OLJCESPC7Z", Quantity: 12}}
	r := Screen(req)
	if !hasReason(r.Reasons, "quantity_anomaly") {
		t.Fatalf("expected quantity_anomaly, got %v", r.Reasons)
	}
}

func TestIncompleteAddress(t *testing.T) {
	req := baseRequest()
	req.ShippingAddress.ZipCode = ""
	r := Screen(req)
	if !hasReason(r.Reasons, "incomplete_address") {
		t.Fatalf("expected incomplete_address, got %v", r.Reasons)
	}
}

func TestMissingCardSignals(t *testing.T) {
	req := baseRequest()
	req.CardBIN = "44"
	r := Screen(req)
	if !hasReason(r.Reasons, "missing_card_signals") {
		t.Fatalf("expected missing_card_signals, got %v", r.Reasons)
	}
}

func TestWorstCaseIsRejected(t *testing.T) {
	req := baseRequest()
	req.Amount.Units = 9000
	req.Items = []Item{{ProductID: "X", Quantity: 50}}
	req.ShippingAddress.ZipCode = ""
	req.ShippingAddress.Country = "ZZ"
	req.CardBIN = ""
	r := Screen(req)
	if r.Decision != DecisionReject {
		t.Fatalf("expected REJECT, got %s (score %d, reasons %v)", r.Decision, r.Score, r.Reasons)
	}
}

func TestScoreNeverExceedsBounds(t *testing.T) {
	req := baseRequest()
	req.Amount.Units = 1 << 40
	req.Items = []Item{{ProductID: "X", Quantity: 1 << 20}}
	req.ShippingAddress = Address{}
	req.CardBIN = ""
	req.CardLast4 = ""
	r := Screen(req)
	if r.Score < 0 || r.Score > 100 {
		t.Fatalf("score out of bounds: %d", r.Score)
	}
}

func hasReason(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}
