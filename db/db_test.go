package db

import (
	"os"
	"testing"
)

func TestCrashRecovery(t *testing.T) {
	os.RemoveAll("sstable")
	os.Remove("wal.log")
	os.MkdirAll("sstable", 0755)

	// write some keys
	store := NewLoremDB(true)
	store.Put("name", "sakar")
	store.Put("lang", "go")

	// simulate crash — don't flush, just create a new instance
	store2 := NewLoremDB(true)

	val, ok := store2.Get("name")
	if !ok || val != "sakar" {
		t.Errorf("expected sakar, got %s", val)
	}
	val, ok = store2.Get("lang")
	if !ok || val != "go" {
		t.Errorf("expected go, got %s", val)
	}

	os.RemoveAll("sstable")
	os.Remove("wal.log")
}
