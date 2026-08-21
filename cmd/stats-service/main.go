package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"supermarket-stats/internal/pocketbase"
	"supermarket-stats/internal/runner"
	"supermarket-stats/internal/scheduler"
)

func main() {
	logger := log.Default()
	logger.Printf("starting supermarket-stats")
	runSupermarket := flag.String("run-supermarket", "", "run one supermarket once")
	runAll := flag.Bool("run-all", false, "run all enabled supermarkets once")
	flag.Parse()
	if *runSupermarket != "" && *runAll { logger.Fatal("--run-supermarket and --run-all cannot be used together") }
	baseURL := os.Getenv("POCKETBASE_URL")
	if baseURL == "" { logger.Fatal("POCKETBASE_URL is required") }
	workers := 8
	if value := os.Getenv("STATS_WORKERS"); value != "" {
		parsed, err := strconv.Atoi(value); if err != nil || parsed < 1 { logger.Fatal("STATS_WORKERS must be a positive integer") }; workers = parsed
	}
	identity := os.Getenv("POCKETBASE_IDENTITY")
	password := os.Getenv("POCKETBASE_PASSWORD")
	if identity == "" || password == "" { logger.Fatal("POCKETBASE_IDENTITY and POCKETBASE_PASSWORD are required") }
	logger.Printf("configuration loaded pocketbase_url=%s workers=%d identity=%s", baseURL, workers, identity)
	client := pocketbase.New(baseURL, identity, password, &http.Client{Timeout: 30 * time.Second})
	repo := pocketbase.NewRepository(client)
	logger.Printf("authenticating PocketBase")
	if err := client.Authenticate(context.Background()); err != nil { logger.Fatalf("authenticate PocketBase: %v", err) }
	logger.Printf("PocketBase authentication successful")
	processRunner := runner.New(repo, runner.Config{Workers: workers, Log: logger})
	if *runSupermarket != "" || *runAll {
		ctx := context.Background()
		if *runSupermarket != "" {
			supermarket, err := repo.Supermarket(ctx, *runSupermarket)
			if err != nil { logger.Fatalf("load supermarket %s: %v", *runSupermarket, err) }
			if err := processRunner.Run(ctx, *supermarket); err != nil { logger.Fatalf("stats %s: %v", supermarket.ID, err) }
			return
		}
		supermarkets, err := repo.Supermarkets(ctx)
		if err != nil { logger.Fatalf("load supermarkets: %v", err) }
		for _, supermarket := range supermarkets {
			if err := processRunner.Run(ctx, supermarket); err != nil { logger.Printf("stats %s: %v", supermarket.ID, err) }
		}
		return
	}
	service := scheduler.New(repo, processRunner, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Printf("service started")
	if err := service.Run(ctx); err != nil && err != context.Canceled { logger.Fatal(err) }
	logger.Printf("service stopped")
}
