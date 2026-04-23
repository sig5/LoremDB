package memtable

import (
	"github.com/google/btree"
)

type MemTableRow struct {
	Key       string
	Value     string
	IsDeleted bool
}

func (row *MemTableRow) Less(than btree.Item) bool {

	return row.Key < than.(*MemTableRow).Key
}

type MemTable struct {
	btree   *btree.BTree
	maxSize int
	size    int
}

func NewMemTable() *MemTable {
	mt := &MemTable{
		btree:   btree.New(32),
		maxSize: 10000,
		size:    0,
	}
	return mt

}

func (table *MemTable) Get(key string) (string, bool) {
	item := table.btree.Get(&MemTableRow{Key: key})
	if item == nil || item.(*MemTableRow).IsDeleted {
		return "", false
	}
	return item.(*MemTableRow).Value, true
}

func (table *MemTable) add(key string, value string, isDeleted bool) bool {
	item := &MemTableRow{
		Key:       key,
		Value:     value,
		IsDeleted: isDeleted,
	}
	table.btree.ReplaceOrInsert(item)
	table.size++
	result := table.size >= table.maxSize
	return result

}

func (table *MemTable) Put(key string, value string) bool {
	return table.add(key, value, false)
}

func (table *MemTable) Delete(key string) bool {
	return table.add(key, "", true)
}

func (table *MemTable) Iterate(fn func(row *MemTableRow)) {

	table.btree.Ascend(func(item btree.Item) bool {
		fn(item.(*MemTableRow))
		return true
	})
}

func (table *MemTable) Size() int {
	return table.size
}
