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

### Sequential reads — hits (10k keys pre-loaded)

| Config | ns/op |
|--------|------:|
| bloom=true, lock=true | 5,445 |
| bloom=true, lock=false | 5,364 |
| bloom=false, lock=true | 5,351 |
| bloom=false, lock=false | 5,333 |

### Sequential reads — misses (10k keys pre-loaded, reading non-existent keys)

| Config | ns/op |
|--------|------:|
| bloom=true, lock=true | 65.4 |
| bloom=true, lock=false | 65.1 |
| bloom=false, lock=true | 67.5 |
| bloom=false, lock=false | 67.4 |

### Reads — large dataset (100k keys, hits vs misses)

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

**Bloom filter only helps misses.** On a hit, the bloom filter says "maybe" and you still have to read the index — so you pay the hash cost for nothing. On a miss, bloom can reject a key outright without touching the index at all, which is where the speedup comes from.

**The bloom advantage scales with index size.** With 10k keys (small SSTables, index fits in cache), bloom saves ~3% on misses (65ns vs 67ns). With 100k keys (larger indices after compaction), the saving grows to ~12% (97ns vs 110ns). The index maps are too large for CPU cache at that point, so each lookup becomes a RAM access; bloom's compact bitset stays cache-hot and wins.

**Bloom never helps read hits.** The hit numbers across small and large datasets are nearly identical with and without bloom (~5.3–5.4µs). The SSTable data file read dominates, and bloom adds a small hash overhead before it.

**An uncontended lock is basically free.** For sequential writes, `lock=true` is actually faster than `lock=false` (58µs vs 73µs). There's no contention so the mutex costs nothing, and the two code paths end up with different allocation patterns — `lock=false` allocates about 30% more memory per op, which is where the slowdown comes from.

**Concurrent writes scale to about 3x, not 10x.** Sequential puts cost ~58µs; parallel drops to ~18µs across 10 goroutines. The ceiling is the memtable write lock — WAL appends and B-tree inserts both serialize, so adding more goroutines doesn't help past a point.

**Concurrent reads barely move the needle** (~5.4µs → 4.7µs). Reads take a shared `RLock` so they can technically run in parallel, but the bottleneck is SSTable file I/O which doesn't fan out well on a single drive.

## Running

```bash
go run .
```
