package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"topdown/internal/replay"
)

// DemoStorage handles persistence of parsed demos
type DemoStorage struct {
	basePath string
}

// DemoMetadata stores lightweight info about a demo
type DemoMetadata struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	MapName    string  `json:"mapName"`
	TickRate   float64 `json:"tickRate"`
	RoundCount int     `json:"roundCount"`
	CreatedAt  string  `json:"createdAt"`
	FileSize   int64   `json:"fileSize"`
}

// NewDemoStorage creates a new storage instance
func NewDemoStorage(basePath string) *DemoStorage {
	return &DemoStorage{basePath: basePath}
}

// SaveReplay saves a replay as protobuf to disk
func (ds *DemoStorage) SaveReplay(demoID string, r *replay.Replay) error {
	demoDir := filepath.Join(ds.basePath, demoID)
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		return fmt.Errorf("failed to create demo directory: %w", err)
	}

	replayPath := filepath.Join(demoDir, "replay.pb")
	return r.SerializeReplayProtobuf(replayPath)
}

// LoadReplay loads a replay from protobuf
func (ds *DemoStorage) LoadReplay(demoID string) (*replay.Replay, error) {
	replayPath := filepath.Join(ds.basePath, demoID, "replay.pb")
	return replay.DeserializeReplayProtobuf(replayPath)
}

// SaveMetadata saves demo metadata as JSON
func (ds *DemoStorage) SaveMetadata(demoID string, metadata *DemoMetadata) error {
	demoDir := filepath.Join(ds.basePath, demoID)
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		return fmt.Errorf("failed to create demo directory: %w", err)
	}

	metadataPath := filepath.Join(demoDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// LoadMetadata loads demo metadata from JSON
func (ds *DemoStorage) LoadMetadata(demoID string) (*DemoMetadata, error) {
	metadataPath := filepath.Join(ds.basePath, demoID, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata DemoMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// ListDemos returns list of all demo IDs and names
func (ds *DemoStorage) ListDemos() ([]DemoMetadata, error) {
	entries, err := os.ReadDir(ds.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []DemoMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read demos directory: %w", err)
	}

	var demos []DemoMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadata, err := ds.LoadMetadata(entry.Name())
		if err != nil {
			continue // Skip demos with invalid metadata
		}

		demos = append(demos, *metadata)
	}

	return demos, nil
}

// DemoExists checks if a demo exists
func (ds *DemoStorage) DemoExists(demoID string) bool {
	demoDir := filepath.Join(ds.basePath, demoID)
	_, err := os.Stat(demoDir)
	return err == nil
}
