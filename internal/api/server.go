package api

import (
	"topdown/internal/replay"
	"topdown/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router      *gin.Engine
	jobQueue    *JobQueue
	storage     *storage.DemoStorage
	currentDemo *replay.Replay
}

// NewServer creates a new API server
func NewServer(demoStoragePath string, numWorkers int) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(cors.Default()) // Enable CORS for all origins. TODO: Configure more securely for production.

	storageService := storage.NewDemoStorage(demoStoragePath)
	jobQueue := NewJobQueue(storageService, numWorkers)

	server := &Server{
		router:   router,
		jobQueue: jobQueue,
		storage:  storageService,
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API endpoints
func (s *Server) setupRoutes() {
	// Demo list and upload
	s.router.GET("/demos", s.ListDemosHandler)
	s.router.POST("/demos", s.UploadDemoHandler)

	// Job status
	s.router.GET("/demos/jobs/:jobId/status", s.GetJobStatusHandler)

	// Demo metadata
	s.router.GET("/demos/:demoId", s.GetDemoHandler)

	// Round data
	s.router.GET("/demos/:demoId/rounds/:roundNum", s.GetRoundHandler)
	s.router.GET("/demos/:demoId/rounds/:roundNum/raw", s.GetRoundRawHandler)
}

// Start starts the API server on the given address
func (s *Server) Start(address string) error {
	return s.router.Run(address)
}
