package stats

import (
	"testing"
	"time"
)

func TestCalculate(t *testing.T) {
	prices := []Price{}
	for i, value := range []float64{1, 1, 2, 2} {
		p := value
		prices = append(prices, Price{Date: time.Date(2026, 8, 18+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), BulkPrice: &p})
	}
	r := Calculate(prices, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC))
	if r.MeanAll.Value != 1.5 || r.ChangesAll.Value != 1 { t.Fatalf("unexpected all stats: %+v", r) }
	if r.MeanPrice[7].Value != 1.5 || r.Changes[7].Value != 1 { t.Fatalf("unexpected window stats: %+v", r) }
}

func TestInvalidSeriesIsEmpty(t *testing.T) {
	zero := 0.0
	r := Calculate([]Price{{Date: "2026-08-21", BulkPrice: &zero}}, time.Now())
	if r.MeanAll.Set || len(r.MeanPrice) != 0 { t.Fatal("invalid series must produce null stats") }
}
