package metadata

import (
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"
)

// InMemoryMetadataStore implements MetadataStore using in-memory storage with periodic persistence
type InMemoryMetadataStore struct {
	config        Config
	files         map[string]*FileMetadata
	nodes         map[string]*NodeInfo
	chunkLocs     map[string][]string
	mu            sync.RWMutex
	lastPersist   time.Time
	persistTicker *time.Ticker
	doneChan      chan struct{}
}

// NewInMemoryMetadataStore creates a new in-memory metadata store
func NewInMemoryMetadataStore(config Config) (*InMemoryMetadataStore, error) {
	store := &InMemoryMetadataStore{
		config:    config,
		files:     make(map[string]*FileMetadata),
		nodes:     make(map[string]*NodeInfo),
		chunkLocs: make(map[string][]string),
		doneChan:  make(chan struct{}),
	}

	// Load persisted data if exists
	if err := store.loadPersistence(); err != nil {
		return nil, err
	}

	// Start persistence ticker if configured
	if config.PersistenceInterval > 0 {
		store.persistTicker = time.NewTicker(config.PersistenceInterval)
		go store.persistenceLoop()
	}

	return store, nil
}

// Close stops the persistence ticker and releases resources
func (s *InMemoryMetadataStore) Close() error {
	if s.persistTicker != nil {
		s.persistTicker.Stop()
	}
	close(s.doneChan)
	return s.savePersistence()
}

// SaveFile saves file metadata
func (s *InMemoryMetadataStore) SaveFile(fileID string, metadata *FileMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata.UpdatedAt = time.Now()
	s.files[fileID] = metadata
	return nil
}

// GetFile retrieves file metadata
func (s *InMemoryMetadataStore) GetFile(fileID string) (*FileMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metadata, exists := s.files[fileID]; exists {
		return metadata, nil
	}
	return nil, ErrNotFound
}

// ListFiles returns all files
func (s *InMemoryMetadataStore) ListFiles() ([]*FileMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files := make([]*FileMetadata, 0, len(s.files))
	for _, file := range s.files {
		files = append(files, file)
	}
	// Sort by creation time for consistent ordering
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt.Before(files[j].CreatedAt)
	})
	return files, nil
}

// DeleteFile removes file metadata
func (s *InMemoryMetadataStore) DeleteFile(fileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.files[fileID]; !exists {
		return ErrNotFound
	}
	delete(s.files, fileID)
	return nil
}

// SaveNode saves node information
func (s *InMemoryMetadataStore) SaveNode(nodeID string, info *NodeInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info.LastHeartbeat = time.Now()
	s.nodes[nodeID] = info
	return nil
}

// GetNode retrieves node information
func (s *InMemoryMetadataStore) GetNode(nodeID string) (*NodeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if node, exists := s.nodes[nodeID]; exists {
		return node, nil
	}
	return nil, ErrNotFound
}

// ListNodes returns all nodes
func (s *InMemoryMetadataStore) ListNodes() ([]*NodeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	// Sort by node ID for consistent ordering
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes, nil
}

// SaveChunkLocations saves the locations where a chunk is stored
func (s *InMemoryMetadataStore) SaveChunkLocations(chunkID string, locations []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chunkLocs[chunkID] = locations
	return nil
}

// GetChunkLocations retrieves the locations where a chunk is stored
func (s *InMemoryMetadataStore) GetChunkLocations(chunkID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if locations, exists := s.chunkLocs[chunkID]; exists {
		// Return a copy to prevent external modification
		result := make([]string, len(locations))
		copy(result, locations)
		return result, nil
	}
	return nil, ErrNotFound
}

// persistenceLoop periodically saves data to disk
func (s *InMemoryMetadataStore) persistenceLoop() {
	for {
		select {
		case <-s.persistTicker.C:
			s.savePersistence()
		case <-s.doneChan:
			return
		}
	}
}

// savePersistence saves the current state to disk
func (s *InMemoryMetadataStore) savePersistence() error {
	if s.config.PersistencePath == "" {
		return nil // No persistence configured
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data := persistenceData{
		Files:     s.files,
		Nodes:     s.nodes,
		ChunkLocs: s.chunkLocs,
	}

	file, err := os.Create(s.config.PersistencePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// loadPersistence loads state from disk
func (s *InMemoryMetadataStore) loadPersistence() error {
	if s.config.PersistencePath == "" {
		return nil // No persistence configured
	}

	if _, err := os.Stat(s.config.PersistencePath); os.IsNotExist(err) {
		return nil // No persistence file yet
	}

	file, err := os.Open(s.config.PersistencePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var data persistenceData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.files = data.Files
	s.nodes = data.Nodes
	s.chunkLocs = data.ChunkLocs
	return nil
}

// persistenceData holds the data to be persisted
type persistenceData struct {
	Files     map[string]*FileMetadata `json:"files"`
	Nodes     map[string]*NodeInfo     `json:"nodes"`
	ChunkLocs map[string][]string      `json:"chunk_locations"`
}
