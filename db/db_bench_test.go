package db

import (
	"fmt"
	"os"
	"testing"
)

func setupDB(b *testing.B, useBloom bool, useLock bool) *LoremDB {
	os.RemoveAll("sstable")
	os.Remove("wal.log")
	os.MkdirAll("sstable", 0755)
	return NewLoremDB(useBloom, useLock)
}

func BenchmarkPut(b *testing.B) {
	configs := []struct {
		useBloom bool
		useLock  bool
	}{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}
	for _, cfg := range configs {
		name := fmt.Sprintf("bloom=%v,lock=%v", cfg.useBloom, cfg.useLock)
		b.Run(name, func(b *testing.B) {
			store := setupDB(b, cfg.useBloom, cfg.useLock)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
			}
		})
	}
}

func BenchmarkGet(b *testing.B) {
	configs := []struct {
		useBloom bool
		useLock  bool
	}{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}
	for _, cfg := range configs {
		name := fmt.Sprintf("bloom=%v,lock=%v", cfg.useBloom, cfg.useLock)
		b.Run(name, func(b *testing.B) {
			store := setupDB(b, cfg.useBloom, cfg.useLock)
			for i := 0; i < 10000; i++ {
				store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.Get(fmt.Sprintf("key-%d", i%10000))
			}
		})
	}
}

// BenchmarkGetMiss reads keys that don't exist. Bloom filter can short-circuit
// every SSTable lookup (no index access needed), so the gap vs no-bloom is largest here.
func BenchmarkGetMiss(b *testing.B) {
	configs := []struct {
		useBloom bool
		useLock  bool
	}{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}
	for _, cfg := range configs {
		name := fmt.Sprintf("bloom=%v,lock=%v", cfg.useBloom, cfg.useLock)
		b.Run(name, func(b *testing.B) {
			store := setupDB(b, cfg.useBloom, cfg.useLock)
			for i := 0; i < 10000; i++ {
				store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.Get(fmt.Sprintf("miss-%d", i))
			}
		})
	}
}

// BenchmarkGetLargeDataset loads enough keys to trigger compaction and produce
// large SSTables whose index maps no longer fit in CPU cache. At this scale
// bloom's compact bitset should beat plain map lookups for both hits and misses.
func BenchmarkGetLargeDataset(b *testing.B) {
	const preload = 100000
	configs := []struct {
		useBloom bool
		label    string
	}{
		{true, "hit,bloom=true"},
		{false, "hit,bloom=false"},
		{true, "miss,bloom=true"},
		{false, "miss,bloom=false"},
	}
	_ = configs
	for _, hit := range []bool{true, false} {
		for _, useBloom := range []bool{true, false} {
			hit, useBloom := hit, useBloom
			label := fmt.Sprintf("hit=%v,bloom=%v", hit, useBloom)
			b.Run(label, func(b *testing.B) {
				store := setupDB(b, useBloom, false)
				for i := 0; i < preload; i++ {
					store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if hit {
						store.Get(fmt.Sprintf("key-%d", i%preload))
					} else {
						store.Get(fmt.Sprintf("miss-%d", i))
					}
				}
			})
		}
	}
}

func BenchmarkPutConcurrent(b *testing.B) {
	configs := []struct {
		useBloom bool
		useLock  bool
	}{
		{true, true},
		{false, true},
	}
	for _, cfg := range configs {
		name := fmt.Sprintf("bloom=%v,lock=%v", cfg.useBloom, cfg.useLock)
		b.Run(name, func(b *testing.B) {
			store := setupDB(b, cfg.useBloom, cfg.useLock)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
					i++
				}
			})
		})
	}
}

func BenchmarkGetConcurrent(b *testing.B) {
	configs := []struct {
		useBloom bool
		useLock  bool
	}{
		{true, true},
		{false, true},
	}
	for _, cfg := range configs {
		name := fmt.Sprintf("bloom=%v,lock=%v", cfg.useBloom, cfg.useLock)
		b.Run(name, func(b *testing.B) {
			store := setupDB(b, cfg.useBloom, cfg.useLock)
			for i := 0; i < 10000; i++ {
				store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					store.Get(fmt.Sprintf("key-%d", i%10000))
					i++
				}
			})
		})
	}
}
