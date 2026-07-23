package persist

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteFileAtomic_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []byte(`{"key":"value"}`)

	if err := WriteFileAtomic(path, data); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(read) != string(data) {
		t.Errorf("expected %q, got %q", data, read)
	}
}

func TestWriteFileAtomic_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	if err := WriteFileAtomic(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("second")); err != nil {
		t.Fatal(err)
	}

	read, _ := os.ReadFile(path)
	if string(read) != "second" {
		t.Errorf("expected 'second', got %q", read)
	}
}

func TestReadFileWithRecovery_Fallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	backup := path + ".bak"

	if err := WriteFileAtomic(backup, []byte("backup-data")); err != nil {
		t.Fatal(err)
	}

	data, err := ReadFileWithRecovery(path)
	if err != nil {
		t.Fatalf("expected recovery, got error: %v", err)
	}
	if string(data) != "backup-data" {
		t.Errorf("expected backup-data, got %q", data)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected backup to be renamed to primary after recovery")
	}
}

func TestStateManager_SnapshotRestore(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	type testState struct {
		Value string `json:"value"`
	}

	sm.Snapshot("test", &testState{Value: "hello"})
	time.Sleep(100 * time.Millisecond)

	var restored testState
	if err := sm.Restore("test", &restored); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if restored.Value != "hello" {
		t.Errorf("expected 'hello', got %q", restored.Value)
	}
}
