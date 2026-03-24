package server

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	Filename string
}

type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
	mu sync.RWMutex
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = CASPathTransformFunc
	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(nodeID, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.PathName)

	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}

func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(nodeID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.PathName)

	return os.RemoveAll(fullPath)
}

func (s *Store) Write(nodeID, key string, r io.Reader) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeStream(nodeID, key, r)
}

func (s *Store) writeStream(nodeID, key string, r io.Reader) (int64, error) {
	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.PathName)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return 0, err
	}

	fullPathWithFilename := filepath.Join(fullPath, pathKey.Filename)

	f, err := os.Create(fullPathWithFilename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return io.Copy(f, r)
}

func (s *Store) WriteDecrypt(encKey []byte, nodeID, key string, r io.Reader) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.PathName)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return 0, err
	}

	fullPathWithFilename := filepath.Join(fullPath, pathKey.Filename)

	f, err := os.Create(fullPathWithFilename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := copyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) Read(nodeID, key string) (int64, io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pathKey := s.PathTransformFunc(key)
	fullPathWithFilename := filepath.Join(s.Root, pathKey.PathName, pathKey.Filename)

	f, err := os.Open(fullPathWithFilename)
	if err != nil {
		return 0, nil, err
	}

	fi, err := os.Stat(fullPathWithFilename)
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), f, nil
}

func CASPathTransformFunc(key string) PathKey {
	hash := hashKey(key)

	// MD5 hash is always 32 characters, but let's be safe
	if len(hash) < 32 {
		// Pad with zeros if hash is too short
		for len(hash) < 32 {
			hash += "0"
		}
	}

	pathName := hash[:28]
	pathName = pathName[:4] + "/" + pathName[4:8] + "/" + pathName[8:12] + "/" + pathName[12:16] + "/" + pathName[16:20] + "/" + pathName[20:24] + "/" + pathName[24:28]
	filename := hash[28:]

	return PathKey{
		PathName: pathName,
		Filename: filename,
	}
}

func init() {
	gob.Register(PathKey{})
}
