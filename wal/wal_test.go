package wal

import (
	"fmt"
	"testing"
)

func TestAppendLog(t *testing.T) {
	path := "wal.log"

	wal, err := NewWal(path)

	if err != nil {
		t.Errorf("Expected no error")
	}

	wal.Append(WalRow{Key: "sakar", Value: "test", IsDeleted: false})
}

func TestRecoverLog(t *testing.T) {
	path := "wal.log"

	wal, err := NewWal(path)

	if err != nil {
		t.Errorf("Expected no error")
	}
	ans := []*WalRow{}

	wal.Recover(func(row *WalRow) {
		ans = append(ans, row)
	})
}

func TestAppendAndRecoverLog(t *testing.T) {
	path := "wal.log"

	wal, err := NewWal(path)

	if err != nil {
		t.Errorf("Expected no error")
	}
	ans := []*WalRow{}

	wal.Append(WalRow{Key: "sakar", Value: "test", IsDeleted: false})

	wal.Recover(func(row *WalRow) {
		ans = append(ans, row)
	})
	for _, row := range ans {
		fmt.Printf("%+v\n", *row)
	}
}
