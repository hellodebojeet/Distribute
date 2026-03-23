package metadata

import (
	"time"
)

// FileMetadata represents the metadata for a file
type FileMetadata struct {
	FileID    string          `json:"file_id"`
	Chunks    []ChunkMetadata `json:"chunks"`
	Version   int64           `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Owner     string          `json:"owner,omitempty"` // for auth
	Size      int64           `json:"size"`            // total size of file
}

// ChunkMetadata represents metadata for a chunk of a file
type ChunkMetadata struct {
	ChunkID   string   `json:"chunk_id"`
	Index     int      `json:"index"` // chunk index in file
	Size      int64    `json:"size"`
	Checksum  string   `json:"checksum"` // SHA256 of chunk data
	Version   int64    `json:"version"`
	Locations []string `json:"locations"` // node IDs that have this chunk
}

// NodeInfo represents information about a storage node
type NodeInfo struct {
	NodeID        string    `json:"node_id"`
	Address       string    `json:"address"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	IsAlive       bool      `json:"is_alive"`
	Chunks        []string  `json:"chunks"`   // chunk IDs stored on this node
	Capacity      int64     `json:"capacity"` // total storage capacity in bytes
	Used          int64     `json:"used"`     // used storage in bytes
}

// ReplicationPlan describes where chunks should be stored
type ReplicationPlan struct {
	FileID  string            `json:"file_id"`
	Chunks  []ChunkAssignment `json:"chunks"`
	Version int64             `json:"version"`
}

// ChunkAssignment describes where a chunk should be stored
type ChunkAssignment struct {
	ChunkID  string   `json:"chunk_id"`
	Primary  string   `json:"primary"`  // primary node ID
	Replicas []string `json:"replicas"` // replica node IDs
}

// MetadataStore defines the interface for metadata persistence
type MetadataStore interface {
	SaveFile(fileID string, metadata *FileMetadata) error
	GetFile(fileID string) (*FileMetadata, error)
	ListFiles() ([]*FileMetadata, error)
	DeleteFile(fileID string) error
	SaveNode(nodeID string, info *NodeInfo) error
	GetNode(nodeID string) (*NodeInfo, error)
	ListNodes() ([]*NodeInfo, error)
	SaveChunkLocations(chunkID string, locations []string) error
	GetChunkLocations(chunkID string) ([]string, error)
}

// Config holds configuration for the metadata service
type Config struct {
	ReplicationFactor   int           `json:"replication_factor"`
	ChunkSize           int64         `json:"chunk_size"` // target chunk size in bytes
	PersistencePath     string        `json:"persistence_path"`
	PersistenceInterval time.Duration `json:"persistence_interval"`
	NodeTimeout         time.Duration `json:"node_timeout"` // time after which node is considered dead
}
