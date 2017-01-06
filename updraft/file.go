package updraft

import "time"

// RevisionID represents a version of a file stored in Updraft.
type RevisionID struct {
	Name string

	// ModTime is the time at which this ModTime was written to Updraft.
	ModTime time.Time
}

type FileMetadata struct {
	RevisionID *RevisionID
	Length     int
	Chunks     []*Chunk
}

// Chunk of a file, including the offset of the chunk, a weak-but-cheap
// checksum value, and a strong hash.
type Chunk struct {
	// Offset is the offset in blocks at which the data for this chunk begins in
	// the file.
	Offset   int
	Checksum []byte
	Hash     []byte
}
