package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type File struct {
	Path string
	Hash string
	Size int64
}

type HashError struct {
	Path string
	Err  error
}

func calculateHash(filePath string, wg *sync.WaitGroup, hashCh chan<- File, errCh chan<- HashError, goroutineCh chan struct{}) {
	defer wg.Done()
	defer func() { <-goroutineCh }()
	goroutineCh <- struct{}{} // Add a goroutine to the channel

	file, err := os.Open(filePath)
	if err != nil {
		errCh <- HashError{Path: filePath, Err: err}
		return
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		errCh <- HashError{Path: filePath, Err: err}
		return
	}

	stat, _ := file.Stat()
	hashCh <- File{Path: filePath, Hash: fmt.Sprintf("%x", hash.Sum(nil)), Size: stat.Size()}
}

func formatPath(path string) string {
	return strings.ReplaceAll(path, `\`, `/`) // Convert Windows paths to Unix-style paths
}

func humanReadableSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	var unitIndex int
	var newSize float64 = float64(size)

	for newSize >= 1024 && unitIndex < len(units)-1 {
		newSize /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", newSize, units[unitIndex])
}

func listFiles(fileMap map[string][]File) {
	for _, files := range fileMap {
		if len(files) > 1 {
			fmt.Printf("Duplicate files with hash %s:\n", files[0].Hash)
			for _, file := range files {
				fmt.Println(file.Path)
			}
			fmt.Println()
		}
	}
}

func moveFiles(fileMap map[string][]File) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Are you sure you want to move duplicated files? (yes/no): ")
	scanner.Scan()
	confirmation := strings.ToLower(scanner.Text())
	if confirmation != "yes" {
		fmt.Println("Move operation canceled.")
		return
	}

	fmt.Print("Enter the destination path to move duplicated files: ")
	scanner.Scan()
	destination := scanner.Text()

	for _, files := range fileMap {
		if len(files) > 1 {
			for i := 1; i < len(files); i++ {
				source := files[i].Path
				dest := filepath.Join(destination, filepath.Base(source))

				// Check if source and destination are on the same disk drive
				srcFileInfo, err := os.Stat(source)
				if err != nil {
					log.Printf("Error getting file info for %s: %v", source, err)
					continue
				}
				dstFileInfo, err := os.Stat(destination)
				if err != nil {
					log.Printf("Error getting file info for %s: %v", destination, err)
					continue
				}

				if os.SameFile(srcFileInfo, dstFileInfo) {
					// Same disk drive, perform a simple rename
					err := os.Rename(source, dest)
					if err != nil {
						log.Printf("Error moving file %s to %s: %v", source, dest, err)
					} else {
						fmt.Printf("Moved file %s to %s\n", source, dest)
					}
				} else {
					// Different disk drives, copy and then delete
					if err := copyFile(source, dest); err != nil {
						log.Printf("Error copying file %s to %s: %v", source, dest, err)
						continue
					}
					if err := os.Remove(source); err != nil {
						log.Printf("Error deleting file %s: %v", source, err)
					} else {
						fmt.Printf("Moved file %s to %s\n", source, dest)
					}
				}
			}
		}
	}
}

// Function to copy a file
func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
}

func deleteFiles(fileMap map[string][]File) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Are you sure you want to delete duplicated files? (yes/no): ")
	scanner.Scan()
	confirmation := strings.ToLower(scanner.Text())
	if confirmation != "yes" {
		fmt.Println("Deletion canceled.")
		return
	}

	for _, files := range fileMap {
		if len(files) > 1 {
			for i := 1; i < len(files); i++ {
				filePath := files[i].Path
				err := os.Remove(filePath)
				if err != nil {
					log.Printf("Error deleting file %s: %v", filePath, err)
				} else {
					fmt.Printf("Deleted file: %s\n", filePath)
				}
			}
		}
	}
}
func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter the folder path to search for duplicates: ")
	scanner.Scan()
	folderPath := formatPath(scanner.Text())

	fileMap := make(map[string][]File)
	var wg sync.WaitGroup
	hashCh := make(chan File)
	errCh := make(chan HashError)
	goroutineCh := make(chan struct{}, runtime.NumCPU()) // Limit the number of concurrently running goroutines
	var fileCount, scannedCount int
	var totalSize int64

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			wg.Add(1)
			go calculateHash(path, &wg, hashCh, errCh, goroutineCh)
			fileCount++
		}
		return nil
	})

	if err != nil {
		log.Fatal("Error:", err)
	}

	go func() {
		wg.Wait()
		close(hashCh)
		close(errCh)
	}()

	fmt.Println("Scanning files...")

	for {
		select {
		case file, ok := <-hashCh:
			if !ok {
				hashCh = nil // Set to nil to exit the loop when both channels are closed
			} else {
				fileMap[file.Hash] = append(fileMap[file.Hash], file)
				scannedCount++
				totalSize += file.Size
				fmt.Printf("\rFiles scanned: %d/%d | Total size: %s | Goroutines: %d/%d", scannedCount, fileCount, humanReadableSize(totalSize), len(goroutineCh), runtime.NumCPU())
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil // Set to nil to exit the loop when both channels are closed
			} else {
				log.Printf("Error processing %s: %v", err.Path, err.Err)
			}
		}

		if hashCh == nil && errCh == nil {
			break // Both channels are closed, exit the loop
		}
	}

	fmt.Println("\nScanning completed.")

	if len(fileMap) > 0 {
		for {
			fmt.Print("Do you want to list, move, delete, or ignore the duplicates? (l/m/d/i): ")
			scanner.Scan()
			action := strings.ToLower(scanner.Text())

			switch action {
			case "l":
				listFiles(fileMap)
			case "m":
				moveFiles(fileMap)
			case "d":
				deleteFiles(fileMap)
			case "i":
				fmt.Println("Duplicates will be ignored.")
				os.Exit(0)
			default:
				fmt.Println("Invalid choice.")
			}
		}
	}
}
