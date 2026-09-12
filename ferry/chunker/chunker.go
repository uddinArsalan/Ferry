package chunker

import (
	"crypto/sha256"
	"io"
	"os"

	"github.com/uddinArsalan/ferry/types"
)

type Chunker struct {
	ChunkLen int
}

func NewChunker(chunkLen int) Chunker {
	return Chunker{ChunkLen: chunkLen}
}

func (c *Chunker) ChunkFile(filePath string) ([]types.Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return []types.Chunk{}, err
	}
	defer file.Close()
	buf := make([]byte, c.ChunkLen*1024)
	chunkIdx := 0
	chunks := make([]types.Chunk, 0)
	for {
		n, err := file.Read(buf)
		if err != nil {
			return []types.Chunk{}, err
		}
		if err == io.EOF {
			break
		}
		chunk := types.Chunk{
			Index: uint32(chunkIdx),
			Hash:  sha256.Sum256(buf[:n]),
			Size:  uint32(len(buf[:n])),
		}
		chunks = append(chunks, chunk)
		chunkIdx++
	}
	return chunks, nil
}

func (c *Chunker) ReadChunk(chunkIdx uint32, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	chunkSize := c.ChunkLen * 1024
	offset := chunkIdx * uint32(chunkSize)
	buf := make([]byte, chunkSize)
	n, err := file.ReadAt(buf, int64(offset))
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
