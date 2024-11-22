package file

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const chunkSize = 10240 // 10 KB chunks

// check if the given file path is valid and accessible
func validator(filepath string) error {
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

// read the file and break it into chunks
func Chunkify(filename string) ([][]byte, error) {
	if err := validator(filename); err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var chunks [][]byte
	buf := make([]byte, chunkSize)

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

// reconstruct the original file from chunks
func Merge(filename string, chunks [][]byte) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	for _, chunk := range chunks {
		_, err := outFile.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk: %v", err)
		}
	}

	return nil
}
