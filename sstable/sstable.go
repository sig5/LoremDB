package sstable

import (
	"bufio"
	"fmt"
	"lorem-lsm/memtable"
	"os"
	"strconv"
	"strings"
)

type SSTable struct {
	indexFile       *os.File
	dataFile        *os.File
	indexFileMemory map[string]int64
}

func CreateSSTable(basePath string) (*SSTable, error) {
	idxFile, err1 := os.OpenFile(fmt.Sprintf("%s/index", basePath), os.O_WRONLY|os.O_CREATE, 0644)

	if err1 != nil {
		return nil, err1
	}

	dFile, err2 := os.OpenFile(fmt.Sprintf("%s/data", basePath), os.O_WRONLY|os.O_CREATE, 0644)

	if err2 != nil {
		return nil, err2
	}

	indexFileMemory := make(map[string]int64)

	return &SSTable{
		indexFile:       idxFile,
		dataFile:        dFile,
		indexFileMemory: indexFileMemory,
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
		indexFileMemory: indexFileMemory,
	}, nil
}

func (ssTable *SSTable) FlushMemTable(memTable *memtable.MemTable) error {
	start := int64(0)
	memTable.Iterate(func(row *memtable.MemTableRow) {
		deleted := "f"
		if row.IsDeleted {
			deleted = "t"
		}

		// write to datafile
		item := fmt.Sprintf("%s,%s,%s\n", row.Key, row.Value, deleted)
		ssTable.dataFile.WriteString(item)
		offset := int64(len(item))
		idxItem := fmt.Sprintf("%s,%d\n", row.Key, start)
		start += offset
		ssTable.indexFile.WriteString(idxItem)
	})
	return nil
}
