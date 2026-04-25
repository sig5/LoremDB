package shard

import (
	"fmt"
	compaction "lorem-lsm/compactor"
	"lorem-lsm/memtable"
	"lorem-lsm/sstable"
	"lorem-lsm/wal"
	"os"
	"sync"
	"time"
)

type LoremDBShard struct {
	wal          *wal.Wal
	memTable     *memtable.MemTable
	ssTables     []*sstable.SSTable
	ssTablePath  string
	useBloom     bool
	useLock      bool
	ssTableLimit int
	lock         sync.RWMutex
	shardId      int
}

func NewLoremDBShard(id int, useBloomShard bool, supportConcurrency bool) *LoremDBShard {

	os.MkdirAll("sstable", 0755)
	writeAheadLog, _ := wal.NewWal(fmt.Sprintf("wal%d.log", id))
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

	return &LoremDBShard{
		wal:          writeAheadLog,
		memTable:     memTable,
		ssTables:     ssTables,
		ssTablePath:  ssTablePath,
		useBloom:     useBloomShard,
		ssTableLimit: 5,
		useLock:      supportConcurrency,
	}
}

func (shard *LoremDBShard) Put(key string, value string) error {
	// write wal first to maximize data recovery chances
	if shard.useLock {
		shard.lock.Lock()
		defer shard.lock.Unlock()
	}

	shard.wal.Append(wal.WalRow{
		Key:       key,
		Value:     value,
		IsDeleted: false,
	})

	isMemTableFull := shard.memTable.Put(key, value)

	if isMemTableFull {
		path := fmt.Sprintf("%s/%d", shard.ssTablePath, time.Now().UnixNano())
		table, err := sstable.CreateSSTable(path, shard.useBloom)

		if err != nil {
			return err
		}

		table.FlushMemTable(shard.memTable)
		shard.ssTables = append(shard.ssTables, table)

		if len(shard.ssTables) > shard.ssTableLimit {
			compactor := compaction.NewCompactor(shard.ssTables)
			shard.ssTables = []*sstable.SSTable{compactor.Compact()}

		}
		shard.memTable = memtable.NewMemTable()
		shard.wal.Clear()
	}
	return nil
}

func (shard *LoremDBShard) Delete(key string) error {

	if shard.useLock {
		shard.lock.Lock()
		defer shard.lock.Unlock()
	}

	// write wal first to maximize data recovery chances
	shard.wal.Append(wal.WalRow{
		Key:       key,
		Value:     "",
		IsDeleted: true,
	})

	isMemTableFull := shard.memTable.Delete(key)

	if isMemTableFull {
		path := fmt.Sprintf("%s/%d", shard.ssTablePath, time.Now().UnixNano())
		table, err := sstable.CreateSSTable(path, shard.useBloom)

		if err != nil {

			return err
		}

		table.FlushMemTable(shard.memTable)
		shard.ssTables = append(shard.ssTables, table)
		shard.memTable = memtable.NewMemTable()
		fmt.Println("reset, new size:", shard.memTable.Size())
	}
	return nil
}

func (shard *LoremDBShard) Get(key string) (string, bool) {

	if shard.useLock {
		shard.lock.RLock()
		defer shard.lock.RUnlock()
	}

	// check in memtable
	val, ok := shard.memTable.Get(key)
	if ok {
		return val, true
	}
	// fallback to sstable
	for i := len(shard.ssTables) - 1; i >= 0; i-- {
		value, _ := shard.ssTables[i].Get(key)

		if value != "" {
			return value, true
		}
	}
	return "", false
}
