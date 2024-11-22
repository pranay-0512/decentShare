package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const ChunkSize = 256000 // 256 KB chunks

// check if the given file path is valid and accessible
func Validator(filepath string) error {
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

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("cannot open file: %v", err)
	}
	defer file.Close()

	return nil
}

// Read the file and break it into chunks
func Chunkify(filename string) ([][]byte, error) {
	if err := Validator(filename); err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var chunks [][]byte
	buf := make([]byte, ChunkSize)

	for {
		bytesRead, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading file: %v", err)
		}

		chunk := make([]byte, bytesRead)
		copy(chunk, buf[:bytesRead])
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// Reconstruct the original file from chunks
func Merge(filename string, chunks [][]byte) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	start := time.Now()
	for i, chunk := range chunks {
		_, err := outFile.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk at index %d: %v", i, err)
		}
	}
	duration := time.Since(start)
	fmt.Printf("File merge completed in %v\n", duration)

	return nil
}
