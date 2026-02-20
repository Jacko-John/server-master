package service

import (
	"fmt"
	"os"
	"path/filepath"
	"server-master/internal/config"
)

type FileService struct {
	cfg *config.Config
}

func NewFileService(cfg *config.Config) *FileService {
	return &FileService{cfg: cfg}
}

func (s *FileService) GetFilePath(filename string) (string, error) {
	// Security check: ensure the filename is just a filename and doesn't contain path traversal
	cleanName := filepath.Base(filename)
	path := filepath.Join(s.cfg.RulePath, cleanName)

	// Check if file exists and is a regular file
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", cleanName)
		}
		return "", fmt.Errorf("failed to access file %s: %w", cleanName, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", cleanName)
	}

	return path, nil
}
