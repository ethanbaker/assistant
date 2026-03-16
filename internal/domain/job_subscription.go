package domain

import (
	"time"

	"gorm.io/gorm"
)

// JobSubscription defines a client's delivery subscription for a job.
type JobSubscription struct {
	*gorm.Model
	Priority        int
	Active          bool
	LastAttemptedAt *time.Time
	LastSuccessAt   *time.Time

	ClientId int
	JobId    int
}
