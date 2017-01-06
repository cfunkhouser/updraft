package updraft

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"
)

const (
	DefaultChunkSize = 4096 // Bytes
)

var (
	ChunkNotFound = errors.New("chunk not found")
	FileNotFound  = errors.New("file not found")
)

type Store interface {
	GetChunk(hash []byte) ([]byte, error)
	GetMetadata(*RevisionID) (*FileMetadata, error)
	GetRevisions(name string) ([]*RevisionID, error)
	ListChunks() [][]byte
	ListFiles() []string
	ListRevisions(name string) ([]*FileMetadata, error)
	PutChunk(hash, data []byte) error
	PutMetadata(*FileMetadata) error
}

// File is a file stored in Updraft. It implements io.Reader.
type File struct {
	fm *FileMetadata
}

type Manager struct {
	chunkSize int
	store     Store
	hash      hash.Hash
}

type ManagerOptions struct {
	ChunkSize int
	HashType  string // TODO(christian): make this work.
}

func NewManager(store Store, opts *ManagerOptions) *Manager {
	hash := sha256.New()
	chunkSize := DefaultChunkSize
	if opts != nil {
		if opts.ChunkSize > 0 {
			chunkSize = opts.ChunkSize
		}
	}
	return &Manager{
		store:     store,
		chunkSize: chunkSize,
		hash:      hash,
	}
}

// Add commits the file readable from f to the contained Store with the metadata
// in rid.
// TODO(christian): Find an approach which allows for concurrent writes for
// Stores which allow it.
func (m *Manager) Add(rid *RevisionID, f io.Reader) error {
	// Write all Chunks to storage. The RevisionID is not written until all
	// Chunks are written successfully. To accomplish this, read all data from f
	// until either the number of bytes read equals chunkSize, or EOF reached.
	fm := &FileMetadata{
		RevisionID: rid,
		Chunks:     make([]*Chunk, 0),
	}
	offset := 0
	for {
		buf := make([]byte, m.chunkSize)
		n, rErr := f.Read(buf)
		if n == 0 {
			break
		}

		chunk, err := m.addChunk(offset, buf[0:n])
		if err != nil {
			return err
		}
		fm.Chunks = append(fm.Chunks, chunk)

		if rErr == io.EOF {
			break
		}
		if rErr != nil {
			return rErr
		}
		offset += 1
	}

	// If we've made it here, there were no errors writing chunks; we can write
	// the metadata now.
	if err := m.store.PutMetadata(fm); err != nil {
		return err
	}
	return nil
}

// Get returns an io.Reader which provides read access to the requested file
// revision. Returns a FileNotFound error under obvious conditions. Using the
// returned reader under such conditions will yield undefined results.
func (m *Manager) Get(rid *RevisionID) (io.Reader, error) {
	// fm, err := m.store.GetMetadata(rid)
	// if err != nil {
	// 	return nil, err
	// }
	return nil, nil
}

// Has reports whether the Store contains a file matching rid.
// TODO(christian): This should succeed only if all chunks also exist.
func (m *Manager) Has(rid *RevisionID) (bool, error) {
	_, err := m.store.GetMetadata(rid)
	if err == FileNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// addChunk calculates the hash for the provided data, commits it to the store,
// and returns the Chunk if the operations are successful.
func (m *Manager) addChunk(offset int, data []byte) (*Chunk, error) {
	m.hash.Reset()
	wn, err := m.hash.Write(data)
	if wn != len(data) {
		return nil, fmt.Errorf("read %v bytes but hashed %v bytes", len(data), wn)
	}
	if err != nil {
		return nil, err
	}
	ch := m.hash.Sum(nil)
	if err := m.store.PutChunk(ch, data); err != nil {
		return nil, err
	}
	return &Chunk{Offset: offset, Hash: ch}, nil
}

// memoryStore implements Store, keeping all data in memory.
type memoryStore struct {
	mu       sync.RWMutex // protects the following members
	chunks   map[string][]byte
	metadata map[string][]*FileMetadata
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{
		chunks:   make(map[string][]byte),
		metadata: make(map[string][]*FileMetadata),
	}
}

func (s *memoryStore) GetChunk(hash []byte) ([]byte, error) {
	s.mu.RLock()
	chunk, ok := s.chunks[fmt.Sprintf("%v", hash)]
	s.mu.RUnlock()

	if !ok {
		return nil, ChunkNotFound
	}
	return chunk, nil
}

func (s *memoryStore) GetMetadata(rid *RevisionID) (*FileMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fms, ok := s.metadata[rid.Name]
	if !ok {
		return nil, FileNotFound
	}
	for _, fm := range fms {
		if fm.RevisionID.ModTime == rid.ModTime {
			n := &(*fm)
			return n, nil
		}
	}
	return nil, FileNotFound // TODO(christian): RevisionNotFound?
}

func (s *memoryStore) GetRevisions(name string) ([]*RevisionID, error) {
	s.mu.RLock()
	r, ok := s.metadata[name]
	s.mu.RUnlock()

	if !ok {
		return nil, FileNotFound
	}
	revisions := make([]*RevisionID, 0)
	for _, fm := range r {
		revisions = append(revisions, fm.RevisionID)
	}
	return revisions, nil
}

func (s *memoryStore) ListChunks() [][]byte {
	chunks := make([][]byte, 0)
	s.mu.RLock()
	defer s.mu.RLock()

	for _, c := range s.chunks {
		chunks = append(chunks, c)
	}
	return chunks
}

func (s *memoryStore) ListFiles() []string {
	names := make([]string, 0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for name := range s.metadata {
		names = append(names, name)
	}
	return names
}

func (s *memoryStore) ListRevisions(name string) ([]*FileMetadata, error) {
	s.mu.RLock()
	revisions, ok := s.metadata[name]
	s.mu.RUnlock()
	if !ok {
		revisions = make([]*FileMetadata, 0)
	}
	return revisions, nil
}

func (s *memoryStore) PutChunk(hash, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks[fmt.Sprintf("%v", hash)] = data
	return nil
}

func (s *memoryStore) PutMetadata(fm *FileMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	md, ok := s.metadata[fm.RevisionID.Name]
	if !ok {
		md = make([]*FileMetadata, 0)
		s.metadata[fm.RevisionID.Name] = md
	}
	md = append(md, fm)
	return nil
}
