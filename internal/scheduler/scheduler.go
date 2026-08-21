package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"supermarket-stats/internal/pocketbase"
	"supermarket-stats/internal/runner"
)

type Scheduler struct {
	repo   *pocketbase.Repository
	runner *runner.Runner
	log    *log.Logger
	mu     sync.Mutex
	running map[string]bool
	lastRun map[string]string
}

func New(repo *pocketbase.Repository, processRunner *runner.Runner, logger *log.Logger) *Scheduler {
	if logger == nil { logger = log.Default() }
	return &Scheduler{repo: repo, runner: processRunner, log: logger, running: map[string]bool{}, lastRun: map[string]string{}}
}

func (s *Scheduler) Run(ctx context.Context) error {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	s.runOnce(ctx, time.Now().Truncate(time.Minute))
	for {
		select {
		case <-ctx.Done(): return ctx.Err()
		case now := <-tick.C: s.runOnce(ctx, now)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context, now time.Time) {
	supermarkets, err := s.repo.Supermarkets(ctx)
	if err != nil { s.log.Printf("load supermarkets: %v", err); return }
	minute := now.Truncate(time.Minute).Format(time.RFC3339)
	for _, supermarket := range supermarkets {
		if supermarket.StatsSchedule == "" { continue }
		cron, err := ParseCron(supermarket.StatsSchedule)
		if err != nil { s.log.Printf("supermarket %s invalid stats_schedule: %v", supermarket.ID, err); continue }
		s.log.Printf("supermarket=%s stats_schedule=%q next_run=%s", supermarket.ID, supermarket.StatsSchedule, cron.Next(now).Format(time.RFC3339))
		if !cron.Matches(now) { continue }
		s.mu.Lock()
		alreadyRun := s.lastRun[supermarket.ID] == minute
		busy := s.running[supermarket.ID]
		if !alreadyRun && !busy { s.lastRun[supermarket.ID] = minute; s.running[supermarket.ID] = true }
		s.mu.Unlock()
		if alreadyRun || busy { continue }
		go func(sm pocketbase.Supermarket) {
			defer func() { s.mu.Lock(); delete(s.running, sm.ID); s.mu.Unlock() }()
			if err := s.runner.Run(ctx, sm); err != nil { s.log.Printf("stats %s: %v", sm.ID, err) }
		}(supermarket)
	}
}
