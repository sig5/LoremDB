package wal

import (
	"bufio"
	"os"
	"strings"
)

type Wal struct {
	file          *os.File
	valueDelimter string
	lineDelimiter string
}

func NewWal(path string) (*Wal, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {

		return nil, err
	}
	return &Wal{
		file:          file,
		lineDelimiter: "\n",
		valueDelimter: ",",
	}, err
}

type WalRow struct {
	Key       string
	Value     string
	IsDeleted bool
}

func (wal *Wal) Append(walRow WalRow) {

	deleted := "f"
	if walRow.IsDeleted {
		deleted = "t"
	}
	wal.file.WriteString(walRow.Key)
	wal.file.WriteString(wal.valueDelimter)
	wal.file.WriteString(walRow.Value)
	wal.file.WriteString(wal.valueDelimter)
	wal.file.WriteString(deleted)
	wal.file.WriteString(wal.lineDelimiter)
}

func (wal *Wal) Clear() error {

	err := wal.file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = wal.file.Seek(0, 0)
	return err
}

func (wal *Wal) Recover(callback func(*WalRow)) error {
	file, err := os.Open(wal.file.Name())

	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, wal.valueDelimter)
		key := parts[0]
		value := parts[1]
		isDeleted := false
		if parts[2] == "t" {
			isDeleted = true
		}

		row := &WalRow{
			Key:       key,
			Value:     value,
			IsDeleted: isDeleted,
		}
		callback(row)
	}

	return nil
}
