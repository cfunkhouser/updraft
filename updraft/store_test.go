package updraft

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// 4x4-byte + 1x2-byte chunks
var testData = []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r'}

// Hash for testData[0:4]
var hash0to4 = []byte{136, 212, 38, 111, 212, 230, 51, 141, 19, 184, 69, 252, 242, 137, 87, 157, 32, 156, 137, 120, 35, 185, 33, 125, 163, 225, 97, 147, 111, 3, 21, 137}

var testRevisionID = &RevisionID{
	Name:    "/foo/bar.txt",
	ModTime: time.Unix(1424703060, 0),
}

func TestManager_addChunk(t *testing.T) {
	memStore := NewMemoryStore()
	manager := NewManager(memStore, &ManagerOptions{ChunkSize: 4})
	if err := manager.Add(testRevisionID, bytes.NewReader(testData)); err != nil {
		t.Fatalf("wanted no error, got: %v", err)
	}

	want := &Chunk{
		Offset: 1,
		Hash:   hash0to4,
	}

	got, err := manager.addChunk(1, testData[0:4])
	if err != nil {
		t.Fatalf("wanted no error, got: %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted: %v, got: %v", want, got)
	}
}

func TestManager_Add(t *testing.T) {
	for _, tt := range []struct {
		chunkSize  int
		wantChunks int
		wantMeta   int
	}{
		{4, 5, 1},
		{2, 9, 1},
		{DefaultChunkSize, 1, 1},
	} {
		store := NewMemoryStore()
		manager := NewManager(store, &ManagerOptions{ChunkSize: tt.chunkSize})
		if err := manager.Add(testRevisionID, bytes.NewReader(testData)); err != nil {
			t.Fatalf("wanted no error, got: %v", err)
		}

		n := 0
		for _ = range store.chunks {
			n += 1
		}
		if n != tt.wantChunks {
			t.Errorf("wanted %v chunks, got %v: %v", tt.wantChunks, n, store.chunks)
		}

		n = 0
		for _ = range store.metadata {
			n += 1
		}
		if n != tt.wantMeta {
			t.Errorf("wanted %v metadata, got %v: %v", tt.wantMeta, n, store.metadata)
		}
	}
}

func TestManager_Has(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, &ManagerOptions{ChunkSize: 4})
	if err := manager.Add(testRevisionID, bytes.NewReader(testData)); err != nil {
		t.Fatalf("wanted no error, got: %v", err)
	}

	fms := []*FileMetadata{
		&FileMetadata{
			RevisionID: &RevisionID{
				Name:    "/foo/bar",
				ModTime: time.Unix(1480716000, 0), // 02 Dec 2016 22:00:00 UTC
			},
			Chunks: []*Chunk{&Chunk{Offset: 0, Hash: hash0to4}},
		},
		&FileMetadata{
			RevisionID: &RevisionID{
				Name:    "/foo/bar",
				ModTime: time.Unix(1424908800, 0), // 26 Feb 2015 00:00:00 UTC
			},
			Chunks: []*Chunk{&Chunk{Offset: 0, Hash: hash0to4}},
		},
		&FileMetadata{
			RevisionID: &RevisionID{
				Name:    "/foo/bar",
				ModTime: time.Unix(1424721060, 0), // 23 Feb 2015 19:51:00 UTC
			},
			Chunks: []*Chunk{&Chunk{Offset: 0, Hash: hash0to4}},
		},
	}
	store.metadata["/foo/bar"] = fms
	store.chunks[fmt.Sprintf("%v", hash0to4)] = testData[0:4]

	for _, tt := range []struct {
		rev  *RevisionID
		want bool
	}{
		{
			rev:  &RevisionID{Name: "/foo/bar", ModTime: time.Unix(1424721060, 0)},
			want: true,
		},
		{
			rev:  &RevisionID{Name: "/foo/bar", ModTime: time.Unix(1424721061, 0)},
			want: false,
		},
	} {
		got, err := manager.Has(tt.rev)
		if err != nil {
			t.Errorf("wanted no errors, got: %v", err)
		}
		if got != tt.want {
			t.Errorf("manager.Has(%v) wanted %v, got %v", tt.rev, tt.want, got)
		}
	}
}
