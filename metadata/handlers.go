package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// ErrNotFound is returned when a resource is not found
var ErrNotFound = errorNotFound{}

type errorNotFound struct{}

func (e errorNotFound) Error() string {
	return "not found"
}

// Handler handles HTTP requests for the metadata service
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
	r.HandleFunc("/nodes", h.handleListNodes).Methods("GET")
	r.HandleFunc("/nodes/{id}", h.handleGetNode).Methods("GET")
	r.HandleFunc("/nodes/{id}", h.handleUpdateNode).Methods("PUT")
	r.HandleFunc("/chunks/{id}/locations", h.handleGetChunkLocations).Methods("GET")
	r.HandleFunc("/chunks/{id}/locations", h.handleUpdateChunkLocations).Methods("PUT")
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

// handleUploadInit handles file upload initialization requests
func (h *Handler) handleUploadInit(w http.ResponseWriter, r *http.Request) {
	var req UploadInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get current file metadata if exists, otherwise create new
	fileMeta, err := h.store.GetFile(req.FileID)
	if err != nil && err != ErrNotFound {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If file doesn't exist, create new metadata
	if err == ErrNotFound {
		fileMeta = &FileMetadata{
			FileID:    req.FileID,
			Chunks:    []ChunkMetadata{},
			Version:   0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Size:      req.Size,
		}
	} else {
		// Update size and version for existing file
		fileMeta.Size = req.Size
		fileMeta.Version++
		fileMeta.UpdatedAt = time.Now()
	}

	// Calculate number of chunks needed
	numChunks := (req.Size + int64(h.config.ChunkSize) - 1) / int64(h.config.ChunkSize)
	if numChunks == 0 {
		numChunks = 1
	}

	// Create chunk assignments
	chunks := make([]ChunkAssignment, numChunks)
	for i := int64(0); i < numChunks; i++ {
		chunkID := fmt.Sprintf("%s_%d", req.FileID, i)

		// Get available nodes for replication (simplified - in production would use consistent hashing)
		nodes, err := h.store.ListNodes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var primary string
		var replicas []string

		if len(nodes) > 0 {
			// Simple assignment: first node as primary, next RF-1 as replicas
			primary = nodes[0].NodeID
			if len(nodes) >= h.config.ReplicationFactor {
				for j := 1; j < h.config.ReplicationFactor && j < len(nodes); j++ {
					replicas = append(replicas, nodes[j].NodeID)
				}
			} else {
				// If not enough nodes, replicate to all available
				for j := 1; j < len(nodes); j++ {
					replicas = append(replicas, nodes[j].NodeID)
				}
			}
		} else {
			// No nodes available - this shouldn't happen in practice
			http.Error(w, "no storage nodes available", http.StatusServiceUnavailable)
			return
		}

		chunks[i] = ChunkAssignment{
			ChunkID:  chunkID,
			Primary:  primary,
			Replicas: replicas,
		}
	}

	// Update file metadata with chunk information
	if fileMeta.Chunks == nil {
		fileMeta.Chunks = make([]ChunkMetadata, numChunks)
	}
	for i := int64(0); i < numChunks; i++ {
		fileMeta.Chunks[i] = ChunkMetadata{
			ChunkID:   chunks[i].ChunkID,
			Index:     int(i),
			Size:      0,  // Will be updated when chunk is committed
			Checksum:  "", // Will be updated when chunk is committed
			Version:   0,
			Locations: []string{}, // Will be updated when chunk is committed
		}
	}

	// Save updated file metadata
	if err := h.store.SaveFile(req.FileID, fileMeta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := UploadInitResponse{
		FileID:  req.FileID,
		Chunks:  chunks,
		Version: fileMeta.Version,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleCommitUpload handles chunk commit requests
func (h *Handler) handleCommitUpload(w http.ResponseWriter, r *http.Request) {
	var req CommitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get file metadata
	fileMeta, err := h.store.GetFile(req.FileID)
	if err != nil {
		if err == ErrNotFound {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find the chunk index
	chunkIndex := -1
	for i, chunk := range fileMeta.Chunks {
		if chunk.ChunkID == req.ChunkID {
			chunkIndex = i
			break
		}
	}
	if chunkIndex == -1 {
		http.Error(w, "chunk not found in file", http.StatusBadRequest)
		return
	}

	// Update chunk metadata with new locations
	chunk := &fileMeta.Chunks[chunkIndex]
	chunk.Locations = req.Locations
	chunk.Version++

	// Update file metadata
	fileMeta.UpdatedAt = time.Now()
	if err := h.store.SaveFile(req.FileID, fileMeta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

// handleUpdateNode updates node information
func (h *Handler) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	var nodeInfo NodeInfo
	if err := json.NewDecoder(r.Body).Decode(&nodeInfo); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.store.SaveNode(nodeID, &nodeInfo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
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
