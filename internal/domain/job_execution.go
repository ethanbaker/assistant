package domain

import "gorm.io/gorm"

// ExecutionStatus represents the lifecycle status of a job execution.
type ExecutionStatus string

const (
	Pending          ExecutionStatus = "PENDING"
	Running          ExecutionStatus = "RUNNING"
	JobFailed        ExecutionStatus = "JOB_FAILED"
	SendingFailed    ExecutionStatus = "SEND_FAILED"
	SuccessfullySent ExecutionStatus = "SUCCESS"
)

// JobExecution stores the output and delivery status for a single job run.
type JobExecution struct {
	*gorm.Model
	Output string
	Status ExecutionStatus
	Error  string

	JobId    int
	ClientId int
}
