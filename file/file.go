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

var TempDst = os.TempDir()

type FileInterface interface {
	GetFileName() string
	GetFilePath() string
	GetFileSize() int64
	GetPieceCount() int
	GetHash() string

	Chunkify() error
	Merge(tempDst string) error
	CalculateHash() (string, error)
	VerifyHash() (bool, error)
	DeleteTempFiles() error
	ReadChunk(chunkIndex int) ([]byte, error)
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

func (f *File) Chunkify() error {
	tempDir := filepath.Join(TempDst, "decent")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	fmt.Println(tempDir)
	file, err := os.Open(f.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var wg sync.WaitGroup
	errChan := make(chan error, f.Pieces)

	for i := range f.Pieces {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			chunkPath := filepath.Join(tempDir, fmt.Sprintf("%s_%d", f.FileName, i))
			chunkFile, err := os.Create(chunkPath)
			if err != nil {
				errChan <- fmt.Errorf("failed to create chunk file %d: %w", i, err)
				return
			}
			defer chunkFile.Close()

			chunk := make([]byte, PieceSize)
			offset := int64(i * PieceSize)

			_, err = file.ReadAt(chunk, offset)
			if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("error reading chunk %d: %w", i, err)
				return
			}

			if i == f.Pieces-1 && f.FileSize%PieceSize != 0 {
				lastPieceSize := f.FileSize % PieceSize
				chunk = chunk[:lastPieceSize]
			}

			if _, err := chunkFile.Write(chunk); err != nil {
				errChan <- fmt.Errorf("failed to write chunk %d: %w", i, err)
				return
			}
		}(i)
	}
	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

func (f *File) Merge(tempDst string) error {
	tempDir := filepath.Join(tempDst, "decent")
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	destDir := filepath.Join(DefaultDestPath, "decent/completed")
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
	for _, file := range files {
		chunkPath := filepath.Join(tempDir, file.Name())
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %s: %w", file.Name(), err)
		}

		_, err = io.Copy(outFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write chunk %s: %w", file.Name(), err)
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
	err = f.DeleteTempFiles()
	return err
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

func (f *File) DeleteTempFiles() error {
	tempDir := filepath.Join(TempDst, "decent")
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	var errChan = make(chan error, len(files))
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file os.DirEntry) {
			defer wg.Done()
			if err := os.Remove(filepath.Join(tempDir, file.Name())); err != nil {
				errChan <- fmt.Errorf("failed to delete temp file %s: %w", file.Name(), err)
				return
			}
		}(file)
	}
	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}
	fmt.Println("Deleted temp files successfully")
	return nil
}

func (f *File) ReadChunk(chunkIndex int) ([]byte, error) {
	chunkPath := filepath.Join(TempDst, "decent", fmt.Sprintf("%s_%d", f.FileName, chunkIndex))
	chunkFile, err := os.Open(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open chunk file %d: %w", chunkIndex, err)
	}
	defer chunkFile.Close()

	chunk := make([]byte, PieceSize)
	_, err = chunkFile.Read(chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk file %d: %w", chunkIndex, err)
	}
	return chunk, nil
}
