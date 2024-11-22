package file

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type FileMerger struct{}

func NewFileMerger() *FileMerger {
	return &FileMerger{}
}

func (fm *FileMerger) Merge(filename string, chunks [][]byte) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var mergeErr error

	start := time.Now()
	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, data []byte) {
			defer wg.Done()

			mu.Lock()
			_, err := outFile.Write(data)
			mu.Unlock()

			if err != nil {
				mu.Lock()
				if mergeErr == nil {
					mergeErr = fmt.Errorf("failed to write chunk at index %d: %v", index, err)
				}
				mu.Unlock()
			}
		}(i, chunk)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("File merge completed in %v\n", duration)

	return mergeErr
}
