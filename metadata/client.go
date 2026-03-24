package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// MetadataClient defines the interface for interacting with the metadata service.
type MetadataClient interface {
	UploadInit(fileID string, size int64) (*UploadInitResponse, error)
	CommitUpload(fileID string, chunkID string, locations []string) error
	GetFileMetadata(fileID string) (*FileMetadata, error)
	ListFiles() ([]*FileMetadata, error)
	DeleteFile(fileID string) error
	GetNode(nodeID string) (*NodeInfo, error)
	ListNodes() ([]*NodeInfo, error)
	GetChunkLocations(chunkID string) ([]string, error)
	SaveChunkLocations(chunkID string, locations []string) error
	MarkUnderReplicated(chunkID string, fileID string) error
}

// HTTPMetadataClient implements MetadataClient using HTTP.
type HTTPMetadataClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPMetadataClient creates a new HTTP metadata client.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	return &HTTPMetadataClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// UploadInitResponse represents the response from the UploadInit endpoint.
type UploadInitResponse struct {
	FileID  string            `json:"file_id"`
	Chunks  []ChunkAssignment `json:"chunks"`
	Version int64             `json:"version"`
}

// UploadInit calls the /init_upload endpoint.
func (c *HTTPMetadataClient) UploadInit(fileID string, size int64) (*UploadInitResponse, error) {
	req := UploadInitRequest{
		FileID: fileID,
		Size:   size,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/init_upload", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var res UploadInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// CommitUpload calls the /commit_upload endpoint.
func (c *HTTPMetadataClient) CommitUpload(fileID string, chunkID string, locations []string) error {
	req := CommitUploadRequest{
		FileID:    fileID,
		ChunkID:   chunkID,
		Locations: locations,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/commit_upload", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}
	return nil
}

// GetFileMetadata calls the /files/{id} endpoint.
func (c *HTTPMetadataClient) GetFileMetadata(fileID string) (*FileMetadata, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/files/" + fileID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var fileMeta FileMetadata
	if err := json.NewDecoder(resp.Body).Decode(&fileMeta); err != nil {
		return nil, err
	}
	return &fileMeta, nil
}

// ListFiles calls the /files endpoint.
func (c *HTTPMetadataClient) ListFiles() ([]*FileMetadata, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/files")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var files []*FileMetadata
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}
	return files, nil
}

// DeleteFile calls the /files/{id} endpoint with DELETE method.
func (c *HTTPMetadataClient) DeleteFile(fileID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/files/"+fileID, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}
	return nil
}

// GetNode calls the /nodes/{id} endpoint.
func (c *HTTPMetadataClient) GetNode(nodeID string) (*NodeInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/nodes/" + nodeID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var nodeInfo NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodeInfo); err != nil {
		return nil, err
	}
	return &nodeInfo, nil
}

// ListNodes calls the /nodes endpoint.
func (c *HTTPMetadataClient) ListNodes() ([]*NodeInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/nodes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var nodes []*NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetChunkLocations calls the /chunks/{id}/locations endpoint.
func (c *HTTPMetadataClient) GetChunkLocations(chunkID string) ([]string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/chunks/" + chunkID + "/locations")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var locations []string
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, err
	}
	return locations, nil
}

// SaveChunkLocations calls the /chunks/{id}/locations endpoint with PUT method.
func (c *HTTPMetadataClient) SaveChunkLocations(chunkID string, locations []string) error {
	reqBody, err := json.Marshal(locations)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/chunks/"+chunkID+"/locations", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}
	return nil
}

// MarkUnderReplicated calls a hypothetical endpoint to mark a chunk as under-replicated.
// In a real implementation, this would trigger background repair processes.
func (c *HTTPMetadataClient) MarkUnderReplicated(chunkID string, fileID string) error {
	// For now, we'll just return nil as this would typically trigger background processes
	// In a full implementation, we might call an endpoint like:
	// POST /chunks/{chunkID}/mark-under-replicated with body {fileID: fileID}
	return nil
}

// UploadInitRequest represents the request body for the UploadInit endpoint.
type UploadInitRequest struct {
	FileID string `json:"file_id"`
	Size   int64  `json:"size"`
}

// CommitUploadRequest represents the request body for the CommitUpload endpoint.
type CommitUploadRequest struct {
	FileID    string   `json:"file_id"`
	ChunkID   string   `json:"chunk_id"`
	Locations []string `json:"locations"`
}
