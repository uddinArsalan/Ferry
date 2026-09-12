package types

type Chunk struct {
	Index uint32
	Hash  [32]byte
	Size  uint32
}

type FileManifest struct {
	Path   string
	Chunks []Chunk
	Size   int64
}
