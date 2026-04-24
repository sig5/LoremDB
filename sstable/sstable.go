package sstable

import (
	"bufio"
	"fmt"
	"lorem-lsm/bloom"
	"lorem-lsm/memtable"
	"os"
	"strconv"
	"strings"
)

type SSTable struct {
	indexFile       *os.File
	dataFile        *os.File
	IndexFileMemory map[string]int64
	bloomFilter     *bloom.BloomFilter
	useBloom        bool
	BasePath        string
}

func (table *SSTable) Close() {

	table.indexFile.Close()
	table.dataFile.Close()
	os.RemoveAll(table.BasePath)
	println("Cleaned")
}

func CreateSSTable(basePath string, useBloom bool) (*SSTable, error) {
	err := os.MkdirAll(basePath, 0755)
	if err != nil {
		return nil, err
	}

	idxFile, err1 := os.OpenFile(fmt.Sprintf("%s/index", basePath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err1 != nil {
		return nil, err1
	}

	dFile, err2 := os.OpenFile(fmt.Sprintf("%s/data", basePath), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

	if err2 != nil {
		return nil, err2
	}

	indexFileMemory := make(map[string]int64)

	return &SSTable{
		indexFile:       idxFile,
		dataFile:        dFile,
		IndexFileMemory: indexFileMemory,
		bloomFilter:     bloom.NewBloomFilter(1000000, 3),
		useBloom:        useBloom,
		BasePath:        basePath,
	}, nil
}

func ReadSSTable(basePath string) (*SSTable, error) {
	idxFile, err1 := os.OpenFile(fmt.Sprintf("%s/index", basePath), os.O_RDONLY, 0644)

	if err1 != nil {
		return nil, err1
	}

	dFile, err2 := os.OpenFile(fmt.Sprintf("%s/data", basePath), os.O_RDONLY, 0644)

	if err2 != nil {
		return nil, err2
	}

	indexFileMemory := make(map[string]int64)
	var data = bufio.NewScanner(idxFile)

	for data.Scan() {
		row := data.Text()
		parts := strings.Split(row, ",")
		offset, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, err
		}
		indexFileMemory[parts[0]] = offset
	}
	return &SSTable{
		indexFile:       idxFile,
		dataFile:        dFile,
		IndexFileMemory: indexFileMemory,
		bloomFilter:     bloom.NewBloomFilter(1000000, 3),
	}, nil
}

func (ssTable *SSTable) FlushMemTable(memTable *memtable.MemTable) error {
	start := int64(0)

	memTable.Iterate(func(row *memtable.MemTableRow) {
		deleted := "f"
		if row.IsDeleted {
			deleted = "t"
		}
		ssTable.bloomFilter.Add(row.Key)
		item := fmt.Sprintf("%s,%s,%s\n", row.Key, row.Value, deleted)
		_, err := ssTable.dataFile.WriteString(item)
		if err != nil {
			fmt.Println("write error:", err)
		}
		ssTable.IndexFileMemory[row.Key] = start
		idxItem := fmt.Sprintf("%s,%d\n", row.Key, start)
		start += int64(len(item))
		ssTable.indexFile.WriteString(idxItem)
	})

	return nil
}

func (ssTable *SSTable) Get(key string) (string, bool) {
	if ssTable.useBloom && !ssTable.bloomFilter.Contains(key) {
		return "", false
	}
	seekPoint, yes := ssTable.IndexFileMemory[key]
	if !yes {
		return "", false
	}

	_, err := ssTable.dataFile.Seek(seekPoint, 0)

	if err != nil {
		return "", false
	}

	reader := bufio.NewReader(ssTable.dataFile)

	row, err := reader.ReadString('\n')
	row = strings.TrimSuffix(row, "\n")
	parts := strings.Split(row, ",")
	isDeleted := parts[2] == "t"
	if isDeleted {
		return "", true
	}
	return parts[1], false
}
