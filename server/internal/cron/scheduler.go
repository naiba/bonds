package cron

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/naiba/bonds/internal/models"
	robfigcron "github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Scheduler struct {
	cron *robfigcron.Cron
	db   *gorm.DB
}

func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		cron: robfigcron.New(robfigcron.WithSeconds()),
		db:   db,
	}
}

func (s *Scheduler) RegisterJob(spec string, name string, fn func()) error {
	_, err := s.cron.AddFunc(spec, func() {
		s.runJob(name, fn)
	})
	if err != nil {
		return fmt.Errorf("failed to register cron job %q: %w", name, err)
	}
	log.Printf("[cron] Registered job %q with spec %q", name, spec)
	return nil
}

func (s *Scheduler) Start() {
	log.Println("[cron] Scheduler started")
	s.cron.Start()
}

func (s *Scheduler) Stop() context.Context {
	log.Println("[cron] Scheduler stopping...")
	return s.cron.Stop()
}

func (s *Scheduler) runJob(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[cron] Job %q panicked: %v", name, r)
		}
	}()

	// Check lock: skip if last run was within 55 seconds
	var cronRecord models.Cron
	result := s.db.Where("command = ?", name).First(&cronRecord)
	if result.Error == nil && cronRecord.LastRunAt != nil {
		if time.Since(*cronRecord.LastRunAt) < 55*time.Second {
			log.Printf("[cron] Job %q skipped: last run %v ago", name, time.Since(*cronRecord.LastRunAt))
			return
		}
	}

	log.Printf("[cron] Job %q starting", name)
	fn()
	log.Printf("[cron] Job %q completed", name)

	// Update LastRunAt
	now := time.Now()
	if result.Error != nil {
		// Record doesn't exist, create it
		s.db.Create(&models.Cron{
			Command:   name,
			LastRunAt: &now,
		})
	} else {
		s.db.Model(&cronRecord).Update("last_run_at", now)
	}
}
