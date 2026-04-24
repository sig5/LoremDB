package compaction

import (
	"fmt"
	"lorem-lsm/memtable"
	"lorem-lsm/sstable"
	"time"
)

type Compactor struct {
	candidates []*sstable.SSTable
	result     *sstable.SSTable
}

func NewCompactor(candidates []*sstable.SSTable) *Compactor {

	path := fmt.Sprintf("sstable/compacted-%d", time.Now().UnixNano())
	result, erro := sstable.CreateSSTable(path, true)
	if erro != nil {
		return nil
	}

	return &Compactor{
		candidates: candidates,
		result:     result,
	}
}

func (compactor *Compactor) Compact() *sstable.SSTable {

	keys := map[string](string){}
	isDeleted := map[string](bool){}

	for _, candidate := range compactor.candidates {

		for key, _ := range candidate.IndexFileMemory {

			keys[key], isDeleted[key] = candidate.Get(key)
		}
	}
	tempMemTable := memtable.NewMemTable()
	for key, val := range keys {
		if !isDeleted[key] {
			tempMemTable.Put(key, val)
		}
	}

	compactor.result.FlushMemTable(tempMemTable)
	compactor.CleanUp()
	return compactor.result
}

func (compactor *Compactor) CleanUp() {

	for _, candidate := range compactor.candidates {
		candidate.Close()
	}

}
