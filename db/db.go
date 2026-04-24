package db

import (
	"fmt"
	compaction "lorem-lsm/compactor"
	"lorem-lsm/memtable"
	"lorem-lsm/sstable"
	"lorem-lsm/wal"
	"os"
	"time"
)

type LoremDB struct {
	wal          *wal.Wal
	memTable     *memtable.MemTable
	ssTables     []*sstable.SSTable
	ssTablePath  string
	useBloom     bool
	ssTableLimit int
}

func NewLoremDB(useBloom bool) *LoremDB {

	os.MkdirAll("sstable", 0755)
	writeAheadLog, _ := wal.NewWal("wal.log")
	memTable := memtable.NewMemTable()

	writeAheadLog.Recover(func(walRow *wal.WalRow) {
		if !walRow.IsDeleted {
			memTable.Put(walRow.Key, walRow.Value)
		} else {
			memTable.Delete(walRow.Key)
		}
	})

	ssTables := []*sstable.SSTable{}
	ssTablePath := "sstable"

	return &LoremDB{
		wal:          writeAheadLog,
		memTable:     memTable,
		ssTables:     ssTables,
		ssTablePath:  ssTablePath,
		useBloom:     useBloom,
		ssTableLimit: 5,
	}
}

func (db *LoremDB) Put(key string, value string) error {
	// write wal first to maximize data recovery chances
	db.wal.Append(wal.WalRow{
		Key:       key,
		Value:     value,
		IsDeleted: false,
	})

	isMemTableFull := db.memTable.Put(key, value)

	if isMemTableFull {
		path := fmt.Sprintf("%s/%d", db.ssTablePath, time.Now().UnixNano())
		table, err := sstable.CreateSSTable(path, db.useBloom)

		if err != nil {
			return err
		}

		table.FlushMemTable(db.memTable)
		db.ssTables = append(db.ssTables, table)

		if len(db.ssTables) > db.ssTableLimit {
			compactor := compaction.NewCompactor(db.ssTables)
			db.ssTables = []*sstable.SSTable{compactor.Compact()}

		}
		db.memTable = memtable.NewMemTable()
		db.wal.Clear()
	}
	return nil
}

func (db *LoremDB) Delete(key string) error {
	// write wal first to maximize data recovery chances
	db.wal.Append(wal.WalRow{
		Key:       key,
		Value:     "",
		IsDeleted: true,
	})

	isMemTableFull := db.memTable.Delete(key)

	if isMemTableFull {
		path := fmt.Sprintf("%s/%d", db.ssTablePath, time.Now().UnixNano())
		table, err := sstable.CreateSSTable(path, db.useBloom)

		if err != nil {

			return err
		}

		table.FlushMemTable(db.memTable)
		db.ssTables = append(db.ssTables, table)
		db.memTable = memtable.NewMemTable()
		fmt.Println("reset, new size:", db.memTable.Size())
	}
	return nil
}

func (db *LoremDB) Get(key string) (string, bool) {
	// check in memtable
	val, ok := db.memTable.Get(key)
	if ok {
		return val, true
	}
	// fallback to sstable
	for i := len(db.ssTables) - 1; i >= 0; i-- {
		value, _ := db.ssTables[i].Get(key)

		if value != "" {
			return value, true
		}
	}
	return "", false
}
