package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type File struct {
	FileName string
	FilePath string // Absolute path
	FileSize int64  // Size in bytes
	Pieces   int    // Number of pieces
	Hash     string // File hash for verification
}

const (
	PieceSize       = 1024 * 1024 // 1MB in bytes
	DefaultDestPath = "C:\\Users\\linkp\\Downloads\\"
)

type FileInterface interface {
	GetFileName() string
	GetFilePath() string
	GetFileSize() int64
	GetPieceCount() int
	GetHash() string

	Chunkify() ([][]byte, error)
	Merge(chunks [][]byte) error
	CalculateHash() (string, error)
	VerifyHash() (bool, error)
}

var _ FileInterface = (*File)(nil)

func NewFile(filePath string) (*File, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	file := &File{
		FilePath: absPath,
		FileName: filepath.Base(absPath),
	}

	size, err := file.size()
	if err != nil {
		return nil, err
	}
	file.FileSize = size
	file.Pieces = file.pieceCount()

	hash, err := file.CalculateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	file.Hash = hash

	return file, nil
}

func (f *File) GetFileName() string {
	return f.FileName
}

func (f *File) GetFilePath() string {
	return f.FilePath
}

func (f *File) GetFileSize() int64 {
	return f.FileSize
}

func (f *File) GetPieceCount() int {
	return f.Pieces
}

func (f *File) GetHash() string {
	return f.Hash
}

func (f *File) size() (int64, error) {
	fi, err := os.Stat(f.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file stats: %w", err)
	}
	return fi.Size(), nil
}

func (f *File) pieceCount() int {
	pieces := f.FileSize / PieceSize
	if f.FileSize%PieceSize != 0 {
		pieces++
	}
	return int(pieces)
}

func (f *File) Chunkify() ([][]byte, error) {
	file, err := os.Open(f.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	chunks := make([][]byte, f.Pieces)
	var wg sync.WaitGroup
	errChan := make(chan error, f.Pieces)

	for i := range f.Pieces {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			chunk := make([]byte, PieceSize)
			offset := int64(index * PieceSize)

			_, err := file.ReadAt(chunk, offset)
			if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("error reading chunk %d: %w", index, err)
				return
			}

			if index == f.Pieces-1 && f.FileSize%PieceSize != 0 {
				lastPieceSize := f.FileSize % PieceSize
				chunk = chunk[:lastPieceSize]
			}

			chunks[index] = chunk
		}(i)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return chunks, nil
}

func (f *File) Merge(chunks [][]byte) error {
	if len(chunks) != f.Pieces {
		return fmt.Errorf("invalid number of chunks: expected %d, got %d", f.Pieces, len(chunks))
	}

	destDir := filepath.Join(DefaultDestPath, "completed")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	outPath := filepath.Join(destDir, f.FileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	start := time.Now()
	for i, chunk := range chunks {
		if _, err := outFile.Write(chunk); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
	}

	mergedHash, err := f.CalculateHash()
	if err != nil {
		return fmt.Errorf("failed to verify merged file: %w", err)
	}

	if mergedHash != f.Hash {
		return fmt.Errorf("file verification failed: hash mismatch")
	}

	fmt.Printf("File merge completed in %v\n", time.Since(start))
	return nil
}

func (f *File) CalculateHash() (string, error) {
	// TODO: Implement file hash calculation
	fmt.Println("Calculating hash for:", f.FilePath)
	return "placeholder-hash", nil
}

func (f *File) VerifyHash() (bool, error) {
	// TODO: Implement file hash verification
	fmt.Println("Verifying hash for:", f.FilePath)
	return true, nil
}
