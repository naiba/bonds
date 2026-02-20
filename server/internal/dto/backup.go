package dto

import "time"

type BackupResponse struct {
	Filename  string    `json:"filename" example:"bonds-2026-02-20-073000.zip"`
	Size      int64     `json:"size" example:"1048576"`
	CreatedAt time.Time `json:"created_at" example:"2026-02-20T07:30:00Z"`
}

type BackupConfigResponse struct {
	CronEnabled   bool   `json:"cron_enabled" example:"true"`
	CronSpec      string `json:"cron_spec" example:"0 0 2 * * *"`
	RetentionDays int    `json:"retention_days" example:"30"`
	BackupDir     string `json:"backup_dir" example:"data/backups"`
	DBDriver      string `json:"db_driver" example:"sqlite"`
}
