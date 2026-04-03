package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListDemosHandler returns all available demos
func (s *Server) ListDemosHandler(c *gin.Context) {
	demos, err := s.storage.ListDemos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"demos": demos})
}

// UploadDemoHandler handles demo file uploads
func (s *Server) UploadDemoHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Generate unique ID for this demo
	demoID := uuid.New().String()

	// Create temp directory if it doesn't exist
	tempDir := "/tmp/topdown-uploads"
	_ = os.MkdirAll(tempDir, 0755)

	// Save uploaded file temporarily
	tempPath := filepath.Join(tempDir, demoID+".dem")
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded file: " + err.Error()})
		return
	}

	filePath := file.Filename
	demoBase := filepath.Base(filePath)
	ext := filepath.Ext(demoBase)
	demoName := strings.TrimSuffix(demoBase, ext)

	// Create and submit parsing job
	job := &ParseJob{
		JobID:     uuid.New().String(),
		DemoID:    demoID,
		DemoName:  demoName,
		DemoPath:  tempPath,
		Status:    JobPending,
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.jobQueue.Submit(job)

	c.JSON(http.StatusAccepted, gin.H{
		"jobId":  job.JobID,
		"demoId": demoID,
		"status": "queued",
	})
}

// GetJobStatusHandler returns the status of a parse job
func (s *Server) GetJobStatusHandler(c *gin.Context) {
	jobID := c.Param("jobId")

	job, err := s.jobQueue.GetStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetDemoHandler returns metadata for a demo
func (s *Server) GetDemoHandler(c *gin.Context) {
	demoID := c.Param("demoId")

	if !s.storage.DemoExists(demoID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Demo not found"})
		return
	}

	metadata, err := s.storage.LoadMetadata(demoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// GetRoundHandler returns round data for a specific round
func (s *Server) GetRoundHandler(c *gin.Context) {
	demoID := c.Param("demoId")
	roundNumStr := c.Param("roundNum")

	roundNum, err := strconv.Atoi(roundNumStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number"})
		return
	}

	if !s.storage.DemoExists(demoID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Demo not found"})
		return
	}

	if s.currentDemo == nil {
		// Load replay from disk
		replayObj, err := s.storage.LoadReplay(demoID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.currentDemo = replayObj
	}

	// // Load replay from disk
	// replayObj, err := s.storage.LoadReplay(demoID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	// Get round data
	roundData, err := s.currentDemo.RoundToJSON(roundNum)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, roundData)
}

// GetRoundRawHandler returns raw round data as JSON string (useful for direct frontend consumption)
func (s *Server) GetRoundRawHandler(c *gin.Context) {
	demoID := c.Param("demoId")
	roundNumStr := c.Param("roundNum")

	roundNum, err := strconv.Atoi(roundNumStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number"})
		return
	}

	if !s.storage.DemoExists(demoID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Demo not found"})
		return
	}

	// Load replay from disk
	replayObj, err := s.storage.LoadReplay(demoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get round data
	roundData, err := replayObj.RoundToJSON(roundNum)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Marshal to JSON for clean output
	jsonData, err := json.Marshal(roundData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", jsonData)
}
