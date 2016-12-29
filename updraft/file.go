package updraft

import "time"

const DefaultChunkSize = 4096 // Bytes

// DefaultFile creates a File with appropriate default values.
func DefaultFile() *File {
	return &File{
		ChunkSize: DefaultChunkSize,
		Chunks:    make([]*Chunk, 0),
	}
}

// File is the metadata representation of a file stored in Updraft.
type File struct {
	Name   string
	Length int

	// ChunkSize in bytes of blocks for this file.
	ChunkSize int
	Chunks    []*Chunk

	// ModifiedTime is the time at which this File was written to Updraft.
	ModifiedTime time.Time
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
