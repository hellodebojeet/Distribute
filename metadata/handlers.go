package metadata

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// hashString returns a hash of the input string
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// ErrNotFound is returned when a resource is not found
var ErrNotFound = errorNotFound{}

type errorNotFound struct{}

func (e errorNotFound) Error() string {
	return "not found"
}

type Handler struct {
	store  MetadataStore
	config Config
}

// NewHandler creates a new metadata service handler
func NewHandler(store MetadataStore, config Config) *Handler {
	return &Handler{
		store:  store,
		config: config,
	}
}

// RegisterRoutes registers all routes with the given router
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/files", h.handleListFiles).Methods("GET")
	r.HandleFunc("/files/{id}", h.handleGetFile).Methods("GET")
	r.HandleFunc("/files", h.handleUploadFile).Methods("POST")
	r.HandleFunc("/files/{id}", h.handleDeleteFile).Methods("DELETE")
	r.HandleFunc("/init_upload", h.handleUploadInit).Methods("POST")
	r.HandleFunc("/commit_upload", h.handleCommitUpload).Methods("POST")
	r.HandleFunc("/nodes", h.handleListNodes).Methods("GET")
	r.HandleFunc("/nodes/{id}", h.handleGetNode).Methods("GET")
	r.HandleFunc("/chunks/{id}/locations", h.handleGetChunkLocations).Methods("GET")
	r.HandleFunc("/chunks/{id}/locations", h.handleUpdateChunkLocations).Methods("PUT")
}

// NewChunkAssignment creates a new chunk assignment
func NewChunkAssignment(chunkID, primary string, replicas []string) ChunkAssignment {
	return ChunkAssignment{
		ChunkID:  chunkID,
		Primary:  primary,
		Replicas: replicas,
	}
}

// handleListFiles returns a list of all files
func (h *Handler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.store.ListFiles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// handleGetFile returns metadata for a specific file
func (h *Handler) handleGetFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	file, err := h.store.GetFile(fileID)
	if err != nil {
		if err == ErrNotFound {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

// handleUploadFile handles file upload requests
func (h *Handler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	var fileMeta FileMetadata
	if err := json.NewDecoder(r.Body).Decode(&fileMeta); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.store.SaveFile(fileMeta.FileID, &fileMeta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileMeta)
}

// handleDeleteFile handles file deletion requests
func (h *Handler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	if err := h.store.DeleteFile(fileID); err != nil {
		if err == ErrNotFound {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListNodes returns a list of all nodes
func (h *Handler) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.store.ListNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

// handleGetNode returns info for a specific node
func (h *Handler) handleGetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	node, err := h.store.GetNode(nodeID)
	if err != nil {
		if err == ErrNotFound {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

// handleUploadInit handles file upload initialization requests
func (h *Handler) handleUploadInit(w http.ResponseWriter, r *http.Request) {
	var req UploadInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.FileID == "" {
		http.Error(w, "file_id is required", http.StatusBadRequest)
		return
	}
	if req.Size <= 0 {
		http.Error(w, "size must be positive", http.StatusBadRequest)
		return
	}

	// Create file metadata
	fileMeta := &FileMetadata{
		FileID:    req.FileID,
		Size:      req.Size,
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.store.SaveFile(req.FileID, fileMeta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate chunk assignments based on chunk size and replication factor
	chunkSize := h.config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 4 * 1024 * 1024 // Default 4MB
	}

	numChunks := int(req.Size / chunkSize)
	if req.Size%chunkSize != 0 {
		numChunks++
	}

	chunks := make([]ChunkAssignment, 0, numChunks)
	nodes, err := h.store.ListNodes()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list nodes: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter only alive nodes
	aliveNodes := make([]*NodeInfo, 0)
	for _, node := range nodes {
		if node.IsAlive {
			aliveNodes = append(aliveNodes, node)
		}
	}

	if len(aliveNodes) == 0 {
		http.Error(w, "no alive nodes available for chunk allocation", http.StatusInternalServerError)
		return
	}

	// Simple hash-based node selection for chunk allocation
	for i := 0; i < numChunks; i++ {
		chunkID := fmt.Sprintf("%s_%d", req.FileID, i)

		// Determine primary node using hash
		primaryIndex := int(hashString(chunkID)) % len(aliveNodes)
		primary := aliveNodes[primaryIndex].NodeID

		// Determine replicas (next N-1 nodes in ring)
		replicas := make([]string, 0)
		repFactor := h.config.ReplicationFactor
		if repFactor <= 0 {
			repFactor = 1
		}
		if repFactor > len(aliveNodes) {
			repFactor = len(aliveNodes)
		}

		for j := 1; j < repFactor; j++ {
			replicaIndex := (primaryIndex + j) % len(aliveNodes)
			replicas = append(replicas, aliveNodes[replicaIndex].NodeID)
		}

		chunks = append(chunks, NewChunkAssignment(chunkID, primary, replicas))
	}

	// Initialize empty chunks in file metadata
	fileMeta.Chunks = make([]ChunkMetadata, len(chunks))
	for i, chunk := range chunks {
		fileMeta.Chunks[i] = ChunkMetadata{
			ChunkID:   chunk.ChunkID,
			Index:     i,
			Size:      0,  // Will be updated when chunk is committed
			Checksum:  "", // Will be updated when chunk is committed
			Version:   1,
			Locations: []string{}, // Will be updated when chunk is committed
		}
	}

	// Update file metadata with chunk info
	if err := h.store.SaveFile(req.FileID, fileMeta); err != nil {
		http.Error(w, fmt.Sprintf("failed to save file metadata: %v", err), http.StatusInternalServerError)
		return
	}

	response := UploadInitResponse{
		FileID:  req.FileID,
		Chunks:  chunks,
		Version: 1,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleCommitUpload handles chunk commit requests
func (h *Handler) handleCommitUpload(w http.ResponseWriter, r *http.Request) {
	var req CommitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.FileID == "" {
		http.Error(w, "file_id is required", http.StatusBadRequest)
		return
	}
	if req.ChunkID == "" {
		http.Error(w, "chunk_id is required", http.StatusBadRequest)
		return
	}
	if len(req.Locations) == 0 {
		http.Error(w, "locations must not be empty", http.StatusBadRequest)
		return
	}

	// Save chunk locations
	if err := h.store.SaveChunkLocations(req.ChunkID, req.Locations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update file metadata with chunk info
	fileMeta, err := h.store.GetFile(req.FileID)
	if err != nil {
		if err == ErrNotFound {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to get file metadata: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the chunk index and update its metadata
	chunkIndex := -1
	for i, chunk := range fileMeta.Chunks {
		if chunk.ChunkID == req.ChunkID {
			chunkIndex = i
			break
		}
	}

	if chunkIndex == -1 {
		http.Error(w, fmt.Sprintf("chunk %s not found in file %s", req.ChunkID, req.FileID), http.StatusBadRequest)
		return
	}

	// Update chunk metadata (we don't have size/checksum here, but we can update locations and version)
	fileMeta.Chunks[chunkIndex].Locations = req.Locations
	fileMeta.Chunks[chunkIndex].Version = fileMeta.Chunks[chunkIndex].Version + 1
	fileMeta.UpdatedAt = time.Now()

	// Save updated file metadata
	if err := h.store.SaveFile(req.FileID, fileMeta); err != nil {
		http.Error(w, fmt.Sprintf("failed to save file metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleGetChunkLocations returns locations for a specific chunk
func (h *Handler) handleGetChunkLocations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	locations, err := h.store.GetChunkLocations(chunkID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locations)
}

// handleUpdateChunkLocations updates locations for a specific chunk
func (h *Handler) handleUpdateChunkLocations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	var locations []string
	if err := json.NewDecoder(r.Body).Decode(&locations); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.store.SaveChunkLocations(chunkID, locations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
