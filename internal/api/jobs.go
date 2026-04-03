package api

import (
	"fmt"
	"os"
	"sync"
	"time"
	"topdown/internal/replay"
	"topdown/internal/storage"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

// JobStatus represents the status of a parse job
type JobStatus string

const (
	JobPending  JobStatus = "pending"
	JobParsing  JobStatus = "parsing"
	JobComplete JobStatus = "complete"
	JobFailed   JobStatus = "failed"
)

// ParseJob represents a demo parsing job
type ParseJob struct {
	JobID       string     `json:"jobId"`
	DemoID      string     `json:"demoId"`
	DemoName    string     `json:"demoName"`
	DemoPath    string     `json:"-"`
	Status      JobStatus  `json:"status"`
	Progress    int        `json:"progress"` // 0-100
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// JobQueue manages async demo parsing
type JobQueue struct {
	jobs    map[string]*ParseJob
	queue   chan *ParseJob
	storage *storage.DemoStorage
	mu      sync.RWMutex
}

// NewJobQueue creates a new job queue with n worker goroutines
func NewJobQueue(storage *storage.DemoStorage, numWorkers int) *JobQueue {
	jq := &JobQueue{
		jobs:    make(map[string]*ParseJob),
		queue:   make(chan *ParseJob, 100),
		storage: storage,
	}

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go jq.worker()
	}

	return jq
}

// Submit adds a job to the queue
func (jq *JobQueue) Submit(job *ParseJob) {
	jq.mu.Lock()
	jq.jobs[job.JobID] = job
	jq.mu.Unlock()

	jq.queue <- job
}

// GetStatus returns the current status of a job
func (jq *JobQueue) GetStatus(jobID string) (*ParseJob, error) {
	jq.mu.RLock()
	defer jq.mu.RUnlock()

	job, exists := jq.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// worker processes jobs from the queue
func (jq *JobQueue) worker() {
	for job := range jq.queue {
		jq.processJob(job)
	}
}

// processJob handles the actual demo parsing
func (jq *JobQueue) processJob(job *ParseJob) {
	jq.updateJobStatus(job.JobID, JobParsing, 0, "")

	// Parse the demo
	f, err := os.Open(job.DemoPath)
	if err != nil {
		jq.updateJobStatus(job.JobID, JobFailed, 0, fmt.Sprintf("Failed to open demo: %v", err))
		return
	}
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	rh := replay.NewReplayHandler(p)

	if err := p.ParseToEnd(); err != nil {
		jq.updateJobStatus(job.JobID, JobFailed, 0, fmt.Sprintf("Failed to parse demo: %v", err))
		return
	}

	jq.updateJobStatus(job.JobID, JobParsing, 50, "")

	// Generate replay
	replayObj := rh.GenerateReplay()

	// Save replay to disk
	if err := jq.storage.SaveReplay(job.DemoID, &replayObj); err != nil {
		jq.updateJobStatus(job.JobID, JobFailed, 0, fmt.Sprintf("Failed to save replay: %v", err))
		return
	}

	jq.updateJobStatus(job.JobID, JobParsing, 80, "")

	// Get demo file name
	// demoBase := filepath.Base(job.DemoPath)
	// ext := filepath.Ext(demoBase)
	// demoName := strings.TrimSuffix(demoBase, ext)

	// Save metadata to disk
	metadata := &storage.DemoMetadata{
		ID:         job.DemoID,
		Name:       job.DemoName,
		MapName:    replayObj.MapName,
		TickRate:   replayObj.TickRate,
		RoundCount: len(replayObj.Rounds),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	// Get file size
	if fileInfo, err := os.Stat(job.DemoPath); err == nil {
		metadata.FileSize = fileInfo.Size()
	}

	if err := jq.storage.SaveMetadata(job.DemoID, metadata); err != nil {
		jq.updateJobStatus(job.JobID, JobFailed, 0, fmt.Sprintf("Failed to save metadata: %v", err))
		return
	}

	now := time.Now()
	jq.updateJobStatus(job.JobID, JobComplete, 100, "")
	jq.mu.Lock()
	jq.jobs[job.JobID].CompletedAt = &now
	jq.mu.Unlock()
}

// updateJobStatus updates the status of a job
func (jq *JobQueue) updateJobStatus(jobID string, status JobStatus, progress int, errMsg string) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if job, exists := jq.jobs[jobID]; exists {
		job.Status = status
		job.Progress = progress
		job.Error = errMsg
		job.UpdatedAt = time.Now()
	}
}
