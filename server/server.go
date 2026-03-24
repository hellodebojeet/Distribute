package server

// MessageReplicateBlob is used to replicate a blob (plaintext) to a specific peer.
type MessageReplicateBlob struct {
	Key  string
	Data []byte
}
