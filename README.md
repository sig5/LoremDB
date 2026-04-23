# lorem-lsm

An LSM-tree key-value store in Go, built from scratch.

## Architecture

Writes go to a WAL first, then into an in-memory B-tree (memtable). When the memtable fills up, it flushes to an SSTable on disk. Reads check the memtable first, then walk SSTables from newest to oldest.

```
Write -> WAL -> Memtable (B-tree)
                     |
                   flush
                     |
               SSTable (disk)

Read -> Memtable -> SSTable[N] -> SSTable[N-1] -> ...
```

| Package | Role |
|---------|------|
| `wal` | Append-only log for crash recovery |
| `memtable` | In-memory B-tree |
| `sstable` | Immutable sorted file on disk |
| `bloom` | Bloom filter to skip SSTables on misses |
| `db` | Coordinates reads and writes |

## Bloom filter

Each SSTable has a bloom filter. On a `Get`, the bloom filter is checked before the index. If the key is definitely absent, the SSTable is skipped.

Whether this actually helps depends on SSTable size. With small SSTables the index map fits in CPU cache, so a map lookup is cheaper than running 3 hash functions. With large SSTables the index spills out of cache and each lookup hits RAM. At that point the bloom filter (a compact bitset that stays cache-hot) is faster at rejecting misses.

## Benchmarks

500,001 writes, 50,000 reads on existing keys, 50,000 reads on missing keys.

**Small SSTables (maxSize=10k, ~50 SSTables)**

| | Writes | Read hits | Read misses |
|-|--------|-----------|-------------|
| Bloom on | 4.19s | 135ms | 58ms |
| Bloom off | 4.16s | 112ms | 39ms |

Bloom is slower. Index fits in cache so map lookups are already fast.

**Large SSTables (maxSize=100k, ~5 SSTables)**

| | Writes | Read hits | Read misses |
|-|--------|-----------|-------------|
| Bloom on | 4.11s | 68ms | 8.5ms |
| Bloom off | 3.91s | 70ms | 12ms |

Bloom wins on misses. Large index maps cause cache pressure and bloom's bitset stays hot.

## Running

```bash
go run .
```
