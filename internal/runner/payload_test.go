package runner

import (
	"testing"
	"time"

	"supermarket-stats/internal/stats"
)

func TestStatsPayloadContainsSchemaFields(t *testing.T) {
	r := stats.Calculate([]stats.Price{{Date: "2026-08-21", BulkPrice: floatPtr(1.234)}}, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC))
	payload := statsPayload("product", r, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC))
	for _, field := range []string{"product_id", "last_updated", "mean_price_d7", "mean_price_d356", "price_diff_mean_all", "price_std_dev_pct_d90", "price_min_all_date", "price_max_all_date", "num_changes_all"} {
		if _, ok := payload[field]; !ok { t.Fatalf("missing payload field %q", field) }
	}
	if payload["mean_price_all"] != 1.23 { t.Fatalf("unexpected rounding: %#v", payload["mean_price_all"]) }
}

func floatPtr(v float64) *float64 { return &v }
