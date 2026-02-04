package mgo

import "time"

type migrateStatus string

const (
	MigrateStatusPending migrateStatus = "pending"
	MigrateStatusRunning migrateStatus = "running"
	MigrateStatusSuccess migrateStatus = "success"
	MigrateStatusFailed  migrateStatus = "failed"
)

type MigrationInfo struct {
	Status  migrateStatus `bson:"status"`
	LastRun time.Time     `bson:"last_run"`
	Error   string        `bson:"error,omitempty"`
	Version int           `bson:"version"`
}
