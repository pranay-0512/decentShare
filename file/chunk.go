package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type FileChunker struct {
	chunkSize int
}

func NewFileChunker(chunkSize int) *FileChunker {
	return &FileChunker{chunkSize: chunkSize}
}

func (fc *FileChunker) Validate(filepath string) error {
	if filepath == "" {
		return errors.New("filepath cannot be empty")
	}

	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filepath)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filepath)
	}

	return nil
}

func (fc *FileChunker) Chunkify(filename string) ([][]byte, error) {
	if err := fc.Validate(filename); err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var chunks [][]byte
	var mu sync.Mutex
	var wg sync.WaitGroup

	for {
		buf := make([]byte, fc.chunkSize)
		bytesRead, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading file: %v", err)
		}

		wg.Add(1)
		go func(chunk []byte) {
			defer wg.Done()
			processedChunk := make([]byte, bytesRead)
			copy(processedChunk, chunk[:bytesRead])

			mu.Lock()
			chunks = append(chunks, processedChunk)
			mu.Unlock()
		}(buf)
	}

	wg.Wait()
	return chunks, nil
}
