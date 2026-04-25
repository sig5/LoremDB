package db

import (
	"lorem-lsm/shard"
	"sync"
)

type LoremDB struct {
	shardCount         int
	dbList             []*shard.LoremDBShard
	supportConcurrency bool
	useBloom           bool
	lock               sync.Mutex
}

func NewLoremDB(shardCount int, useBloom bool, supportConcurrency bool) *LoremDB {

	return &LoremDB{
		shardCount:         shardCount,
		dbList:             make([]*shard.LoremDBShard, shardCount),
		useBloom:           useBloom,
		supportConcurrency: supportConcurrency,
	}
}

func (db *LoremDB) Put(key string, value string) error {
	shard := db.GetShard(&key)
	return shard.Put(key, value)
}

func (db *LoremDB) Get(key string) (string, bool) {
	shard := db.GetShard(&key)
	return shard.Get(key)
}

func (db *LoremDB) Delete(key string) error {
	shard := db.GetShard(&key)
	return shard.Delete(key)
}

func (db *LoremDB) GetShard(key *string) *shard.LoremDBShard {
	id := db.GetShardId(key)
	selectedShard := db.dbList[id]
	if selectedShard == nil {
		// lazy loadingl
		db.lock.Lock()
		defer db.lock.Unlock()
		selectedShard = db.dbList[id]
		if selectedShard == nil {
			selectedShard = shard.NewLoremDBShard(id, db.useBloom, db.supportConcurrency)
			db.dbList[id] = selectedShard
		}
	}
	return selectedShard
}

func (db *LoremDB) GetShardId(key *string) int {

	// get value of first 5 characters.
	sum := 0

	for i := 0; i < min(len(*key), 5); i++ {
		sum += int((*key)[i] - 'a')
	}
	shardHash := sum % (db.shardCount)
	return shardHash
}
