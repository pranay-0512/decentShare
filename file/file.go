package file

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"p2p/config"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type File struct {
	FileName  string
	FilePath  string     // Absolute path
	FileSize  int64      // Size in bytes
	Pieces    int        // Number of pieces
	Hash      [20]byte   // File hash for verification
	PieceHash [][20]byte // Piece hashes for verification
}

const (
	PieceSize = 1024 * 1024 // 1MB in bytes
)

var (
	DefaultDestPath = config.GetDestPath()
	TempDst         = os.TempDir()
	tempDir         = filepath.Join(TempDst, "decent")
)

type FileInterface interface {
	GetFileName() string
	GetFilePath() string
	GetFileSize() int64
	GetPieceCount() int
	GetHash() [20]byte

	Chunkify(errChan chan error)
	Merge(tempDst string, errChan chan error)
	CalculateHash()
	VerifyHash() bool
	DeleteTempFiles(errChan chan error)
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
	file.PieceHash = make([][20]byte, file.Pieces)

	file.CalculateHash()

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

func (f *File) GetHash() [20]byte {
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

func (f *File) Chunkify(errChan chan error) {
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		errChan <- fmt.Errorf("failed to create temp directory: %w", err)
	}
	fmt.Println(tempDir)
	file, err := os.Open(f.FilePath)
	if err != nil {
		errChan <- fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var wg sync.WaitGroup

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

			if i == f.Pieces-1 && f.FileSize%PieceSize != 0 {
				lastPieceSize := f.FileSize % PieceSize
				chunk = chunk[:lastPieceSize]
			}

			n, err := file.ReadAt(chunk, offset)
			if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("error reading chunk %d: %w", i, err)
				return
			}
			fmt.Println("chunk size: ", len(chunk))
			fmt.Println("read bytes: ", n)

			f.CalculatePieceHash(chunk, i)

			if _, err := chunkFile.Write(chunk); err != nil {
				errChan <- fmt.Errorf("failed to write chunk %d: %w", i, err)
				return
			}
		}(i)
	}
	wg.Wait()

	f.CalculateHash()
	if len(errChan) > 0 {
		for range errChan {
			fmt.Println("error: ", <-errChan)
		}
	}
	fmt.Println("initial file hash: ", f.Hash)
	fmt.Println()
	for i := range f.GetPieceCount() {
		fmt.Print("piece: ", i, " ")
		fmt.Println("piece hash: ", f.PieceHash[i])
	}
	fmt.Println()
}

func (f *File) Merge(tempDst string, errChan chan error) {
	files, err := os.ReadDir(tempDir)
	if err != nil {
		errChan <- fmt.Errorf("failed to read temp directory: %w", err)
		return
	}

	if err := os.MkdirAll(config.GetDestPath(), 0755); err != nil {
		errChan <- fmt.Errorf("failed to create destination directory: %w", err)
	}

	outPath := filepath.Join(config.GetDestPath(), f.FileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		errChan <- fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	start := time.Now()
	for i, file := range files {
		chunkPath := filepath.Join(tempDir, file.Name())
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			errChan <- fmt.Errorf("failed to open chunk file %s: %w", file.Name(), err)
		}
		chunkFileInfo, err := chunkFile.Stat()
		if err != nil {
			errChan <- fmt.Errorf("failed to get chunk file stats: %w", err)
		}
		chunk := make([]byte, int(chunkFileInfo.Size()))
		n, err := chunkFile.Read(chunk)
		f.VerifyPieceHash(chunk, i)
		_, err = io.Copy(outFile, chunkFile)
		chunkFile.Close()
		fmt.Println("piece index: ", i)
		fmt.Println("chunk size: ", len(chunk))
		fmt.Println("read bytes: ", n)
		fmt.Println()
		if err != nil {
			errChan <- fmt.Errorf("failed to write chunk %s: %w", file.Name(), err)
		}
	}

	isValid := f.VerifyHash()
	if !isValid {
		errChan <- fmt.Errorf("failed to verify merged file")
	}
	fmt.Println()
	fmt.Println("merged file hash: ", f.Hash)
	fmt.Println()
	fmt.Printf("File merge completed in %v\n", time.Since(start))
	f.DeleteTempFiles(errChan)
	fmt.Println("check dest path", config.GetDestPath())
}

func (f *File) CalculateHash() {
	hash := sha1.Sum([]byte(f.FileName + strconv.Itoa(f.Pieces) + strconv.Itoa(int(f.FileSize))))
	f.Hash = hash
}

func (f *File) VerifyHash() bool {
	hash := sha1.Sum([]byte(f.FileName + strconv.Itoa(f.Pieces) + strconv.Itoa(int(f.FileSize))))
	if hash == f.Hash {
		return true
	}
	return false
}

func (f *File) CalculatePieceHash(piece []byte, index int) {
	hash := sha1.Sum(piece)
	f.PieceHash[index] = hash
}

func (f *File) VerifyPieceHash(piece []byte, index int) bool {
	hash := sha1.Sum(piece)
	fmt.Println("piece hash: ", hash)
	return hash == f.PieceHash[index]
}

func (f *File) DeleteTempFiles(errChan chan error) {
	files, err := os.ReadDir(tempDir)
	if err != nil {
		errChan <- fmt.Errorf("failed to read temp directory: %w", err)
	}

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

	fmt.Println("Deleted temp files successfully")
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
