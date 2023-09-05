package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Helper function to create a temporary directory for testing and return its path
func createTempDirForTest(t *testing.T) string {
	tempDir, err := ioutil.TempDir(".", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	return tempDir
}

func TestFormatPath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"C:\\path\\to\\file", "C:/path/to/file"},
		{"D:\\documents\\file.txt", "D:/documents/file.txt"},
		{"/unix-style/path", "/unix-style/path"},
		{"relative/path", "relative/path"},
		{"C:/folder/with spaces", "C:/folder/with spaces"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := formatPath(tc.input)
			if result != tc.expected {
				t.Errorf("Expected: %s, Got: %s", tc.expected, result)
			}
		})
	}
}

func TestHumanReadableSize(t *testing.T) {
	testCases := []struct {
		size     int64
		expected string
	}{
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := humanReadableSize(tc.size)
			if result != tc.expected {
				t.Errorf("Expected: %s, Got: %s", tc.expected, result)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary test directory
	tempDir := createTempDirForTest(t)
	defer os.RemoveAll(tempDir)

	// Create source and destination file paths within the temporary directory
	sourceFile := filepath.Join(tempDir, "test_source.txt")
	destFile := filepath.Join(tempDir, "test_dest.txt")

	// Write some content to the source file
	content := []byte("Test content")
	err := ioutil.WriteFile(sourceFile, content, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test the copyFile function
	err = copyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("Error copying file: %v", err)
	}

	// Check if the destination file exists and has the same content
	copiedContent, err := ioutil.ReadFile(destFile)
	if err != nil {
		t.Errorf("Error reading destination file: %v", err)
	}

	if string(copiedContent) != string(content) {
		t.Errorf("Copied content does not match source content")
	}
}

func TestCalculateHash(t *testing.T) {
	// Create a temporary test directory
	tempDir := createTempDirForTest(t)
	var wg sync.WaitGroup
	defer os.RemoveAll(tempDir)

	// Create test files with different content and sizes within the temporary directory
	testFiles := []struct {
		path    string
		content []byte
	}{
		{filepath.Join(tempDir, "same_test_file1.txt"), []byte("Test content 1")},
		{filepath.Join(tempDir, "test_file2.txt"), []byte("Test content 2")},
		{filepath.Join(tempDir, "test_file3.txt"), []byte("Test content 3")},
		{filepath.Join(tempDir, "same_test_file4.txt"), []byte("Test content 1")},
		{filepath.Join(tempDir, "same_test_file5.txt"), []byte("Test content 1")},
	}

	for _, file := range testFiles {
		err := ioutil.WriteFile(file.path, file.content, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test the calculateHash function for each test file
	hashCh := make(chan File)
	errCh := make(chan HashError)
	goroutineCh := make(chan struct{}, 1)

	for _, file := range testFiles {
		wg.Add(1)
		go calculateHash(file.path, &wg, hashCh, errCh, goroutineCh)
	}
	fileMap := make(map[string]int)
	for range testFiles {
		select {
		case file := <-hashCh:
			fileMap[file.Hash] = fileMap[file.Hash] + 1
		case err := <-errCh:
			t.Errorf("Error calculating hash: %v", err.Err)
		}
	}

	for k, file := range fileMap {
		if strings.EqualFold(k, "9c192053ffbc363705b13508c36566f6") && file != 3 {
			t.Fatal("Wrong size found!")
		} else if !strings.EqualFold(k, "9c192053ffbc363705b13508c36566f6") && file != 1 {
			t.Fatal("Wrong size found!")

		}
	}

}

func TestMoveFiles(t *testing.T) {
	// Create a temporary test directory
	tempDir := createTempDirForTest(t)
	defer os.RemoveAll(tempDir)

	tempDir2 := createTempDirForTest(t)
	defer os.RemoveAll(tempDir2)

	// Create test files with different content and sizes within the temporary directory
	testFiles := []struct {
		sourcePath string
		destPath   string
		content    []byte
	}{
		{filepath.Join(tempDir, "test_source1.txt"), filepath.Join(tempDir2, "test_source1.txt"), []byte("Test content 1")},
		{filepath.Join(tempDir, "test_source2.txt"), filepath.Join(tempDir2, "test_source2.txt"), []byte("Test content 2")},
		{filepath.Join(tempDir, "test_source3.txt"), filepath.Join(tempDir2, "test_source3.txt"), []byte("Test content 3")},
	}

	for _, file := range testFiles {
		err := ioutil.WriteFile(file.sourcePath, file.content, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test the moveFiles function for each test file
	fileMap := make(map[string][]File)

	for _, file := range testFiles {
		fileMap["hash123"] = append(fileMap["hash123"], File{Path: file.sourcePath, Hash: "hash123", Size: int64(len(file.content))})
	}

	moveFiles(fileMap, tempDir2)

	// Check if the files were moved to their respective destination paths
	for idx, file := range testFiles {
		if idx == 0 {
			continue // first file should not be moved
		}
		_, err := os.Stat(file.destPath)
		if err != nil {
			t.Errorf("Error moving file: %v", err)
		}
	}
}

func TestDeleteFiles(t *testing.T) {
	// Create a temporary test directory
	tempDir := createTempDirForTest(t)
	defer os.RemoveAll(tempDir)

	// Create test files with different content and sizes within the temporary directory
	testFiles := []struct {
		path    string
		content []byte
	}{
		{filepath.Join(tempDir, "test_file1.txt"), []byte("Test content 1")},
		{filepath.Join(tempDir, "test_file2.txt"), []byte("Test content 2")},
		{filepath.Join(tempDir, "test_file3.txt"), []byte("Test content 3")},
	}

	for _, file := range testFiles {
		err := ioutil.WriteFile(file.path, file.content, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test the deleteFiles function for each test file
	fileMap := make(map[string][]File)

	for _, file := range testFiles {
		fileMap["hash123"] = append(fileMap["hash123"], File{Path: file.path, Hash: "hash123", Size: int64(len(file.content))})
	}

	deleteFiles(fileMap, true)

	// Check if the files were deleted
	for idx, file := range testFiles {
		if idx == 0 {
			continue // first file should not be moved
		}
		_, err := os.Stat(file.path)
		if !os.IsNotExist(err) {
			t.Errorf("Error deleting file: %v", err)
		}
	}
}

// Add more tests for other functions as needed

func TestMain(m *testing.M) {
	// Run tests
	exitCode := m.Run()

	// Exit with the code from the tests
	os.Exit(exitCode)
}
