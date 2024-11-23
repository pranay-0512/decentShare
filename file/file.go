package file

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type File struct {
	FileName string
	FilePath string // -> absolute path
	FileSize int64  // -> bytes
	Pieces   int    // -> number of pieces
}

const (
	PieceSize int    = 1024000 // -> 1MB
	DestPath  string = "C:/Users/linkp/OneDrive/Desktop/decentShare/"
)

func (file *File) Size() (int64, error) {
	fi, err := os.Stat(file.FilePath)
	if err != nil {
		return 0, nil
	}
	size := fi.Size()
	return size, nil
}

func (file *File) PieceCount() int {
	size := file.FileSize
	pieces := size / int64(PieceSize)
	if size%int64(PieceSize) != 0 {
		pieces++
	}
	return int(pieces)
}

func NewFile(FilePath string) (*File, error) {
	file := &File{
		FilePath: FilePath,
	}
	file.FileName = file.FilePath[strings.LastIndex(file.FilePath, "/")+1:]
	size, err := file.Size()
	if err != nil {
		return nil, err
	}
	file.FileSize = size

	file.Pieces = file.PieceCount()

	return file, nil
}

func (file *File) Chunkify() ([][]byte, error) {
	fi, err := os.Open(file.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer fi.Close()

	chunks := make([][]byte, file.Pieces)
	var wg sync.WaitGroup

	for i := range file.Pieces {
		buf := make([]byte, PieceSize)
		bytesRead, err := fi.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading file: %v", err)
		}

		wg.Add(1)
		go func(index int, chunk []byte) {
			defer wg.Done()
			processedChunk := make([]byte, bytesRead)
			copy(processedChunk, chunk[:bytesRead])

			chunks[index] = processedChunk
		}(i, buf)
	}
	wg.Wait()
	return chunks, nil
}

func (file *File) Merge(chunks [][]byte) error {
	outFile, err := os.Create(DestPath + file.FileName)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	var mergeErr error
	start := time.Now()
	for i, chunk := range chunks {
		_, err := outFile.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk at index %d: %v", i, err)
		}
	}
	duration := time.Since(start)
	fmt.Printf("File merge completed in: %v\n", duration)

	return mergeErr
}
