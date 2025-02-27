// Package jobs define an interface to execute jobs
package jobs

// JobIDNotional is the job id for notional job.
const (
	JobIDNotional       = "JOB_NOTIONAL_USD"
	JobIDTransferReport = "JOB_TRANSFER_REPORT"
)

// Job is the interface for jobs.
type Job interface {
	Run() error
}
