package metadata

import (
	"encoding/json"
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

	// Generate chunk assignments (simplified)
	chunks := make([]ChunkAssignment, 0)
	// In a real implementation, this would split the file into chunks
	// and assign them to nodes based on the replication factor

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

	// Save chunk locations
	if err := h.store.SaveChunkLocations(req.ChunkID, req.Locations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
