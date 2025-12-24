package parser

import (
	"context"
	"fmt"
	"time"

	"github.com/kedr891/cs-parser/pkg/logger"
)

type Scheduler struct {
	service         *Service
	intervalMinutes int
	log             *logger.Logger
}

func NewScheduler(service *Service, intervalMinutes int, log *logger.Logger) *Scheduler {
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}

	return &Scheduler{
		service:         service,
		intervalMinutes: intervalMinutes,
		log:             log,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.log.Info("Scheduler started", "interval_minutes", s.intervalMinutes)

	if err := s.runParsingCycle(ctx); err != nil {
		s.log.Error("Initial parsing cycle failed", "error", err)
	}

	ticker := time.NewTicker(time.Duration(s.intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Scheduler stopped by context")
			return ctx.Err()

		case <-ticker.C:
			if err := s.runParsingCycle(ctx); err != nil {
				s.log.Error("Parsing cycle failed", "error", err)
			}
		}
	}
}

func (s *Scheduler) runParsingCycle(ctx context.Context) error {
	s.log.Info("Starting parsing cycle")
	startTime := time.Now()

	if err := s.service.ParseAllSkins(ctx); err != nil {
		return fmt.Errorf("parse all skins: %w", err)
	}

	duration := time.Since(startTime)
	s.log.Info("Parsing cycle completed",
		"duration", duration.String(),
		"next_run", time.Now().Add(time.Duration(s.intervalMinutes)*time.Minute).Format(time.RFC3339),
	)

	return nil
}

func (s *Scheduler) RunOnce(ctx context.Context) error {
	return s.runParsingCycle(ctx)
}

func (s *Scheduler) RunDiscovery(ctx context.Context, query string) error {
	s.log.Info("Starting discovery", "query", query)
	startTime := time.Now()

	if err := s.service.DiscoverNewSkins(ctx, query); err != nil {
		return fmt.Errorf("discover new skins: %w", err)
	}

	duration := time.Since(startTime)
	s.log.Info("Discovery completed", "duration", duration.String())

	return nil
}

func (s *Scheduler) ScheduledDiscovery(ctx context.Context, queries []string, intervalHours int) error {
	s.log.Info("Starting scheduled discovery",
		"queries", len(queries),
		"interval_hours", intervalHours,
	)

	for _, query := range queries {
		if err := s.service.DiscoverNewSkins(ctx, query); err != nil {
			s.log.Error("Discovery failed", "query", query, "error", err)
		}
		time.Sleep(5 * time.Second)
	}

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Discovery scheduler stopped")
			return ctx.Err()

		case <-ticker.C:
			for _, query := range queries {
				if err := s.service.DiscoverNewSkins(ctx, query); err != nil {
					s.log.Error("Discovery failed", "query", query, "error", err)
				}
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (s *Scheduler) GetStats(ctx context.Context) (*SchedulerStats, error) {
	parserStats, err := s.service.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get parser stats: %w", err)
	}

	return &SchedulerStats{
		IsRunning:       true,
		IntervalMinutes: s.intervalMinutes,
		TotalSkins:      parserStats.TotalSkins,
		LastParseTime:   parserStats.LastParseTime,
		RequestsLastMin: parserStats.RequestsLastMin,
	}, nil
}

type SchedulerStats struct {
	IsRunning       bool      `json:"is_running"`
	IntervalMinutes int       `json:"interval_minutes"`
	TotalSkins      int       `json:"total_skins"`
	LastParseTime   time.Time `json:"last_parse_time"`
	RequestsLastMin int       `json:"requests_last_min"`
}

func (s *Scheduler) StartWithDiscovery(ctx context.Context, discoveryQueries []string, discoveryIntervalHours int) error {
	go func() {
		if err := s.Start(ctx); err != nil {
			s.log.Error("Parser scheduler error", "error", err)
		}
	}()

	if len(discoveryQueries) > 0 && discoveryIntervalHours > 0 {
		return s.ScheduledDiscovery(ctx, discoveryQueries, discoveryIntervalHours)
	}

	<-ctx.Done()
	return ctx.Err()
}
