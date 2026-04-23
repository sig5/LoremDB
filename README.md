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

## Running

```bash
go run .
```
