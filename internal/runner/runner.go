package runner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"supermarket-stats/internal/pocketbase"
	"supermarket-stats/internal/stats"
)

type Config struct {
	Workers int
	Now     func() time.Time
	Log     *log.Logger
}

type Runner struct {
	repo *pocketbase.Repository
	cfg  Config
}

type ProductError struct {
	Stage     string `json:"stage"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	EntityID  string `json:"entity_id,omitempty"`
}

type runDetails struct {
	SupermarketID  string `json:"supermarket_id"`
	ProductsTotal  int    `json:"products_total"`
	ProductsDone   int    `json:"products_processed"`
	ProductsFailed int    `json:"products_failed"`
	Workers        int    `json:"worker_count"`
	DurationMS     int64  `json:"duration_ms"`
	ProductsSaved  int    `json:"products_saved"`
	ProductsPerSec float64 `json:"throughput_items_per_second"`
	PricesLoaded   int    `json:"prices_loaded"`
	AvgProductMS   int64  `json:"avg_product_duration_ms"`
	MaxProductMS   int64  `json:"max_product_duration_ms"`
}

func New(repo *pocketbase.Repository, cfg Config) *Runner {
	if cfg.Workers < 1 { cfg.Workers = 8 }
	if cfg.Now == nil { cfg.Now = time.Now }
	if cfg.Log == nil { cfg.Log = log.Default() }
	return &Runner{repo: repo, cfg: cfg}
}

func jobSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n").Replace(value)
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' { b.WriteRune(r) } else if b.Len() > 0 { b.WriteByte('_') }
	}
	return strings.Trim(b.String(), "_")
}

func completedErrors() []ProductError {
	// ponytail: PocketBase currently rejects []; remove this marker when the field accepts empty arrays.
	return []ProductError{{Code: "job_completed", Message: "Job completed", Stage: "complete"}}
}

func (r *Runner) Run(ctx context.Context, supermarket pocketbase.Supermarket) error {
	started := time.Now()
	r.cfg.Log.Printf("stats job started supermarket=%s", supermarket.ID)
	products, err := r.repo.Products(ctx, supermarket.ID)
	if err != nil { return err }
	details := runDetails{SupermarketID: supermarket.ID, ProductsTotal: len(products), Workers: r.cfg.Workers}
	job := &pocketbase.Job{}
	if err := r.repo.CreateJob(ctx, map[string]any{"type": "stats:" + jobSlug(supermarket.Name), "status": "running", "start_date": r.cfg.Now().UTC().Format(time.RFC3339), "details": map[string]any{"schema_version": 1}, "errors": []ProductError{{Code: "job_started", Message: "Job started", Stage: "start"}}}, job); err != nil { return err }

	errors := []ProductError{}
	var metricsMu sync.Mutex
	var productDuration time.Duration
	var maxProductDuration time.Duration
	var pricesLoaded int
	jobs := make(chan pocketbase.Product)
	var wg sync.WaitGroup
	var mu sync.Mutex
	worker := func() {
		defer wg.Done()
		for product := range jobs {
			if ctx.Err() != nil { return }
			productStarted := time.Now()
			prices, err := r.processProduct(ctx, product)
			elapsed := time.Since(productStarted)
			metricsMu.Lock()
			productDuration += elapsed
			if elapsed > maxProductDuration { maxProductDuration = elapsed }
			pricesLoaded += prices
			metricsMu.Unlock()
			if elapsed >= time.Second { r.cfg.Log.Printf("slow product supermarket=%s product=%s duration_ms=%d", supermarket.ID, product.ID, elapsed.Milliseconds()) }
			if err != nil {
				mu.Lock(); errors = append(errors, ProductError{Stage: "process", Code: "product_error", Message: err.Error(), Retryable: true, EntityID: product.ID}); details.ProductsFailed++; mu.Unlock()
			} else { mu.Lock(); details.ProductsDone++; mu.Unlock() }
		}
	}
	workers := r.cfg.Workers
	if workers > len(products) { workers = len(products) }
	if workers == 0 {
		details.DurationMS = time.Since(started).Milliseconds()
		r.cfg.Log.Printf("stats job finished supermarket=%s duration_ms=%d processed=0 failed=0 products_per_second=0 prices=0", supermarket.ID, details.DurationMS)
		return r.repo.UpdateJob(ctx, job.ID, map[string]any{"status": "completed", "end_date": r.cfg.Now().UTC().Format(time.RFC3339), "details": details, "errors": completedErrors()})
	}
	for i := 0; i < workers; i++ { wg.Add(1); go worker() }
	for _, product := range products {
		select { case jobs <- product: case <-ctx.Done(): close(jobs); wg.Wait(); _ = r.repo.UpdateJob(context.Background(), job.ID, map[string]any{"status": "failed", "end_date": r.cfg.Now().UTC().Format(time.RFC3339), "details": details, "errors": []ProductError{{Code: "context_cancelled", Message: ctx.Err().Error(), Stage: "run"}}}); return ctx.Err() }
	}
	close(jobs)
	wg.Wait()

	status := "completed"
	if len(errors) > 0 { status = "completed_with_errors" }
	duration := time.Since(started)
	details.DurationMS = duration.Milliseconds()
	details.ProductsSaved = details.ProductsDone
	details.PricesLoaded = pricesLoaded
	details.MaxProductMS = maxProductDuration.Milliseconds()
	if details.ProductsDone+details.ProductsFailed > 0 {
		details.AvgProductMS = (productDuration / time.Duration(details.ProductsDone+details.ProductsFailed)).Milliseconds()
	}
	if duration > 0 { details.ProductsPerSec = float64(details.ProductsDone) / duration.Seconds() }
	r.cfg.Log.Printf("stats job finished supermarket=%s duration_ms=%d processed=%d failed=%d products_per_second=%.2f prices=%d", supermarket.ID, details.DurationMS, details.ProductsDone, details.ProductsFailed, details.ProductsPerSec, details.PricesLoaded)
	if len(errors) == 0 { errors = completedErrors() }
	if err := r.repo.UpdateJob(ctx, job.ID, map[string]any{"status": status, "end_date": r.cfg.Now().UTC().Format(time.RFC3339), "details": details, "errors": errors}); err != nil { return err }
	return nil
}

func (r *Runner) processProduct(ctx context.Context, product pocketbase.Product) (int, error) {
	prices, err := r.repo.Prices(ctx, product.ID)
	if err != nil { return 0, fmt.Errorf("fetch prices: %w", err) }
	series := make([]stats.Price, len(prices))
	for i, price := range prices { series[i] = stats.Price{Date: price.Date, BulkPrice: price.BulkPrice} }
	if len(prices) == 0 {
		return 0, fmt.Errorf("calculate: no price history")
	}
	for _, price := range prices {
		if price.BulkPrice == nil || *price.BulkPrice <= 0 {
			return len(prices), fmt.Errorf("calculate: invalid bulk_price on %s", price.Date)
		}
	}
	result := stats.Calculate(series, r.cfg.Now())
	data := statsPayload(product.ID, result, r.cfg.Now())
	existing, err := r.repo.FindStats(ctx, product.ID)
	if err != nil { return len(prices), fmt.Errorf("find products_stats: %w", err) }
	if existing == nil { return len(prices), r.repo.CreateStats(ctx, data) }
	return len(prices), r.repo.UpdateStats(ctx, existing.ID, data)
}
