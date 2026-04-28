package cron

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	robfigcron "github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const minRunInterval = 55 * time.Second

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

	acquired, err := s.acquireLock(name)
	if err != nil {
		log.Printf("[cron] Job %q lock error: %v", name, err)
		return
	}
	if !acquired {
		log.Printf("[cron] Job %q skipped: another instance ran it recently or holds the lock", name)
		return
	}

	log.Printf("[cron] Job %q starting", name)
	fn()
	log.Printf("[cron] Job %q completed", name)
}

// acquireLock claims exclusive permission to run a job. Postgres uses
// pg_try_advisory_xact_lock so two replicas firing at the same instant cannot
// both observe a stale last_run_at; SQLite relies on its writer serialisation.
func (s *Scheduler) acquireLock(name string) (bool, error) {
	if s.db.Dialector.Name() == "postgres" {
		return s.acquireLockPostgres(name)
	}
	return s.acquireLockGeneric(name)
}

func (s *Scheduler) acquireLockGeneric(name string) (bool, error) {
	threshold := time.Now().Add(-minRunInterval)

	res := s.db.Model(&models.Cron{}).
		Where("command = ? AND (last_run_at IS NULL OR last_run_at < ?)", name, threshold).
		Update("last_run_at", time.Now())
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 1 {
		return true, nil
	}

	now := time.Now()
	err := s.db.Create(&models.Cron{Command: name, LastRunAt: &now}).Error
	if err == nil {
		return true, nil
	}
	if isUniqueConstraintErr(err) {
		return false, nil
	}
	return false, err
}

func (s *Scheduler) acquireLockPostgres(name string) (bool, error) {
	var acquired bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var got bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", jobLockKey(name)).Scan(&got).Error; err != nil {
			return err
		}
		if !got {
			return nil
		}

		threshold := time.Now().Add(-minRunInterval)
		res := tx.Model(&models.Cron{}).
			Where("command = ? AND (last_run_at IS NULL OR last_run_at < ?)", name, threshold).
			Update("last_run_at", time.Now())
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 1 {
			acquired = true
			return nil
		}

		now := time.Now()
		if err := tx.Create(&models.Cron{Command: name, LastRunAt: &now}).Error; err != nil {
			if isUniqueConstraintErr(err) {
				return nil
			}
			return err
		}
		acquired = true
		return nil
	})
	return acquired, err
}

func jobLockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("bonds:cron:"))
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}

func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "violates unique constraint")
}
