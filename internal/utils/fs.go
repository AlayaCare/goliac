package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
)

func Exists(fs billy.Filesystem, path string) (bool, error) {
	_, err := fs.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ReadFile(fs billy.Filesystem, path string) ([]byte, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func WriteFile(fs billy.Filesystem, filename string, data []byte, perm os.FileMode) error {
	f, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

// RemoveAll function to recursively delete directories and files
func RemoveAll(fs billy.Filesystem, path string) error {
	// Read the contents of the directory
	infos, err := fs.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, no need to remove
		}
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Iterate over each file and subdirectory
	for _, info := range infos {
		fullPath := filepath.Join(path, info.Name())

		if info.IsDir() {
			// Recursively remove the subdirectory
			err = RemoveAll(fs, fullPath)
			if err != nil {
				return fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
			}
		} else {
			// Remove the file
			err = fs.Remove(fullPath)
			if err != nil {
				return fmt.Errorf("failed to remove file %s: %w", fullPath, err)
			}
		}
	}

	// Finally, remove the now-empty directory
	err = fs.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", path, err)
	}

	return nil
}
