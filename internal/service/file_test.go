package service

import (
	"os"
	"path/filepath"
	"server-master/internal/config"
	"testing"
)

func TestFileService_GetFilePath(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a test file
	testFileName := "test.yaml"
	testFilePath := filepath.Join(tempDir, testFileName)
	if err := os.WriteFile(testFilePath, []byte("payload: []"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a subdirectory
	subDirName := "subdir"
	if err := os.Mkdir(filepath.Join(tempDir, subDirName), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	cfg := &config.Config{
		RulePath: tempDir,
	}
	s := NewFileService(cfg)

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "valid file",
			filename: testFileName,
			wantErr:  false,
		},
		{
			name:     "non-existent file",
			filename: "missing.yaml",
			wantErr:  true,
		},
		{
			name:     "is a directory",
			filename: subDirName,
			wantErr:  true,
		},
		{
			name:     "path traversal attempt",
			filename: "../secret.txt", // filepath.Base will turn this into secret.txt
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := s.GetFilePath(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				expectedPath := filepath.Join(tempDir, tt.filename)
				if path != expectedPath {
					t.Errorf("GetFilePath() = %v, want %v", path, expectedPath)
				}
			}
		})
	}
}
