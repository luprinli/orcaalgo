package persist

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteFileAtomic(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, ".orca_tmp_")
	if err != nil {
		return fmt.Errorf("atomic write: create temp: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("atomic write: write temp: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("atomic write: sync temp: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomic write: close temp: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomic write: rename: %w", err)
	}
	return nil
}

func ReadFileWithRecovery(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err == nil {
		return data, nil
	}

	backupFile := filename + ".bak"
	backupData, backupErr := os.ReadFile(backupFile)
	if backupErr == nil {
		os.Rename(backupFile, filename)
		return backupData, nil
	}

	return nil, fmt.Errorf("read recovery failed: primary=%w, backup=%w", err, backupErr)
}
