package runner

import (
	"fmt"
	"math"
	"time"

	"supermarket-stats/internal/stats"
)

func statsPayload(productID string, result stats.Result, now time.Time) map[string]any {
	data := map[string]any{"product_id": productID, "last_updated": now.UTC().Format(time.RFC3339)}
	put := func(name string, value stats.Value) {
		if value.Set {
			data[name] = round(value.Value)
		} else {
			data[name] = nil
		}
	}
	for _, d := range stats.Windows {
		s := fmt.Sprint(d)
		put("mean_price_d"+s, result.MeanPrice[d])
		put("price_diff_mean_d"+s, result.DiffMean[d])
		put("price_diff_mean_pct_d"+s, result.DiffMeanPct[d])
		put("price_std_dev_d"+s, result.StdDev[d])
		put("price_std_dev_pct_d"+s, result.StdDevPct[d])
		put("price_min_d"+s, result.Min[d])
		put("price_max_d"+s, result.Max[d])
		put("num_changes_d"+s, result.Changes[d])
	}
	put("mean_price_all", result.MeanAll)
	put("price_diff_mean_all", result.DiffMeanAll)
	put("price_diff_mean_pct_all", result.DiffMeanPctAll)
	put("price_min_all", result.MinAll)
	put("price_max_all", result.MaxAll)
	put("num_changes_all", result.ChangesAll)
	data["price_min_all_date"] = nullableDate(result.MinAllDate)
	data["price_max_all_date"] = nullableDate(result.MaxAllDate)
	return data
}

func nullableDate(value string) any {
	if value == "" { return nil }
	return value
}

func round(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) { return 0 }
	return math.Round(value*100) / 100
}
