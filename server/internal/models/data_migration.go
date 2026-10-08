package models

// DataMigration records a completed, transactional data conversion. It contains
// no user data; its presence prevents reinterpreting records created afterwards.
type DataMigration struct {
	Name string `gorm:"primaryKey;type:text"`
}
