# LoremDB

An LSM-tree key-value store built in Go from scratch.

## How it works

Every write goes to the WAL first (so nothing is lost on a crash), then into the memtable. When the memtable is full it gets flushed to disk as an SSTable. Reads check the memtable first, then go through SSTables newest to oldest until the key is found.

```
Write -> WAL -> Memtable (B-tree)
                     |
                   flush
                     |
               SSTable (disk)

Read -> Memtable -> SSTable[N] -> SSTable[N-1] -> ...
```

## Memtable

Keys are stored in a B-tree so they stay sorted. This makes flushing cheap since the SSTable is just a sequential write of an already-sorted tree, no extra sorting pass needed.

Deletes don't remove the key, they write a tombstone (`IsDeleted=true`). The key disappears from reads but the tombstone sticks around until compaction cleans it up. Compaction isn't implemented yet.

## SSTable

Each flush creates a directory with two files:

- `data` — every row as `key,value,deleted\n`, written in key order
- `index` — `key,byte_offset\n` for every row in data

The index is kept in memory as a map. A read looks up the offset, seeks to that spot in the data file, and reads one line. SSTables never change after they're written, so newer SSTables always win over older ones on reads.

## Bloom filter

Each SSTable has a bloom filter so reads can skip SSTables that definitely don't have the key without touching the index.

Interestingly, whether this actually helps depends on how big the SSTables are. With small SSTables the whole index fits in CPU cache so a map lookup is faster than hashing the key 3 times. With large SSTables the index is too big for cache and each lookup has to go to RAM. At that point the bloom filter wins because its bitset is small enough to stay cache-hot.

## Benchmarks

500,001 writes, 50,000 reads on keys that exist, 50,000 on keys that don't.

**Small SSTables (maxSize=10k, ~50 SSTables)**

| | Writes | Read hits | Read misses |
|-|--------|-----------|-------------|
| Bloom on | 4.19s | 135ms | 58ms |
| Bloom off | 4.16s | 112ms | 39ms |

Bloom is slower here. The index fits in cache so a plain map lookup is cheaper than running 3 hashes.

**Large SSTables (maxSize=100k, ~5 SSTables)**

| | Writes | Read hits | Read misses |
|-|--------|-----------|-------------|
| Bloom on | 4.11s | 68ms | 8.5ms |
| Bloom off | 3.91s | 70ms | 12ms |

Bloom wins on misses. Index maps are too big for cache, bloom's bitset isn't.

**Without compaction (maxSize=10k, ~50 small SSTables)**

| | Writes | Read hits | Read misses |
|-|--------|-----------|-------------|
| Bloom on | 4.25s | 125ms | 52ms |
| Bloom off | 4.44s | 114ms | 34ms |

Reads walk 50 SSTables. Each index fits in CPU cache so map lookups are fast and bloom adds more overhead than it saves.

**With compaction (maxSize=100k, ~5 large SSTables)**

| | Writes | Read hits | Read misses |
|-|--------|-----------|-------------|
| Bloom on | 15.4s | 62ms | 8.3ms |
| Bloom off | 15.7s | 68ms | 9.1ms |

Fewer SSTables so reads are faster. Writes are slower since each flush is much larger. Index maps no longer fit in CPU cache, so bloom's compact bitset beats a plain map lookup on misses.

## go test benchmarks

Run on Apple M4, `-benchtime=5s`. Numbers are per-operation.

### Sequential writes

| Config | ns/op | B/op |
|--------|------:|-----:|
| bloom=true, lock=true | 58,165 | 28,180 |
| bloom=true, lock=false | 73,167 | 36,454 |
| bloom=false, lock=true | 70,526 | 35,228 |
| bloom=false, lock=false | 80,792 | 39,588 |

### Sequential reads, hits (10k keys pre-loaded)

| Config | ns/op |
|--------|------:|
| bloom=true, lock=true | 5,445 |
| bloom=true, lock=false | 5,364 |
| bloom=false, lock=true | 5,351 |
| bloom=false, lock=false | 5,333 |

### Sequential reads, misses (10k keys pre-loaded, reading non-existent keys)

| Config | ns/op |
|--------|------:|
| bloom=true, lock=true | 65.4 |
| bloom=true, lock=false | 65.1 |
| bloom=false, lock=true | 67.5 |
| bloom=false, lock=false | 67.4 |

### Reads, large dataset (100k keys, hits vs misses)

| Config | ns/op |
|--------|------:|
| hit, bloom=true | 5,418 |
| hit, bloom=false | 5,378 |
| miss, bloom=true | 97.5 |
| miss, bloom=false | 110.4 |

### Concurrent writes (`b.RunParallel`, 10 goroutines)

| Config | ns/op |
|--------|------:|
| bloom=true, lock=true | 18,639 |
| bloom=false, lock=true | 18,889 |

### Concurrent reads (`b.RunParallel`, 10 goroutines)

| Config | ns/op |
|--------|------:|
| bloom=true, lock=true | 4,696 |
| bloom=false, lock=true | 4,754 |

### What the numbers tell us

**Bloom filter only helps misses.** On a hit, bloom says "maybe" and you still have to read the index, so you pay the hash cost for nothing. On a miss it can reject the key without touching the index at all, which is where the speedup comes from.

**The advantage scales with index size.** With 10k keys the SSTable index maps fit in CPU cache, so a plain map lookup beats running 3 hashes and bloom only saves about 3% on misses (65ns vs 67ns). With 100k keys the indices spill out of cache and each lookup costs a RAM round-trip. At that point bloom's compact bitset stays cache-hot and the saving grows to about 12% (97ns vs 110ns).

**Bloom never helps hits.** Hit latency is nearly identical with and without bloom across both dataset sizes (~5.3-5.4µs). The SSTable data file read dominates and bloom just adds a small hash overhead before it.

**An uncontended lock is basically free.** Sequential writes with `lock=true` are actually faster than `lock=false` (58µs vs 73µs). There's no contention so the mutex costs nothing, and the two code paths end up with different allocation patterns. `lock=false` allocates about 30% more per op, which is where the difference comes from.

**Concurrent writes scale to about 3x, not 10x.** Sequential puts cost ~58µs; parallel drops to ~18µs across 10 goroutines. WAL appends and B-tree inserts both serialize on the write lock, so more goroutines can't help past that point.

**Concurrent reads barely move.** ~5.4µs drops to ~4.7µs. Reads take a shared `RLock` so they can run in parallel, but the bottleneck is SSTable file I/O which doesn't fan out well on a single drive.

## Sharding

A single-shard LSM is a single lock. Every write and read globally serializes on one `sync.RWMutex`. Under concurrency that's a bottleneck — goroutines queue up waiting for the lock even when they'd be operating on completely different keys.

Sharding splits the database into N independent shards, each with its own memtable, SSTable list, WAL, and lock. A key goes to exactly one shard, determined by a hash. Goroutines hitting different shards never contend.

```
Key → Hash → Shard ID → LoremDBShard (own lock, WAL, memtable, SSTables)
```

### How the hash works

```go
func (db *LoremDB) GetShardId(key *string) int {
    sum := 0
    for i := 0; i < min(len(*key), 5); i++ {
        sum += int((*key)[i] - 'a')
    }
    return sum % db.shardCount
}
```

Sum the ASCII offset from `'a'` for the first 5 characters, mod shard count.

### Lazy initialization

Shards aren't created at startup. The first key that hashes to a shard ID triggers creation. Double-checked locking makes this safe under concurrency: check without lock, lock, check again, create if still nil.

### Benchmarks

Run on Apple M4, `-benchtime=5s`, 10 goroutines (`b.RunParallel`).

#### Concurrent Gets

| Config | ns/op | vs. 1 shard |
|--------|------:|------------:|
| bloom=true, shardCount=1 | 1,577 | baseline |
| bloom=false, shardCount=1 | 1,530 | baseline |
| bloom=true, shardCount=5 | 812 | **1.9× faster** |
| bloom=false, shardCount=5 | 676 | **2.3× faster** |

#### Concurrent Puts

| Config | ns/op | vs. 1 shard |
|--------|------:|------------:|
| bloom=true, shardCount=1 | 11,365 | baseline |
| bloom=false, shardCount=1 | 12,344 | baseline |
| bloom=true, shardCount=2 | 9,003 | **1.3× faster** |
| bloom=false, shardCount=2 | 10,700 | **1.2× faster** |

### What the numbers tell us

**Reads benefit more than writes.** Gets at 5 shards are ~2× faster; Puts at 2 shards are ~1.3× faster. Reads take a shared `RLock` so multiple goroutines on the same shard proceed in parallel. Writes take an exclusive lock and append to the WAL, which is sequential per shard.

**At shardCount=5, bloom=false beats bloom=true on Gets (676 vs 812 ns/op).** This is the same pattern seen in the single-shard small-SSTable benchmarks — fewer keys per shard means smaller index maps and cheaper plain lookups than running bloom hashes.

## Running

```bash
go run .
```
