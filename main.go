package main

import (
	"fmt"
	"lorem-lsm/db"
	"time"
)

type BenchmarkResult struct {
	useBloom   bool
	writes     time.Duration
	readHits   time.Duration
	readMisses time.Duration
}

func main() {
	withBloom := run_benchmark(true)
	withoutBloom := run_benchmark(false)

	for _, r := range []BenchmarkResult{withBloom, withoutBloom} {
		fmt.Printf("--- bloom=%v ---\n", r.useBloom)
		fmt.Printf("  writes:      %v\n", r.writes)
		fmt.Printf("  read hits:   %v\n", r.readHits)
		fmt.Printf("  read misses: %v\n", r.readMisses)
	}
}

func run_benchmark(useBloom bool) BenchmarkResult {
	store := db.NewLoremDB(useBloom, true)

	start := time.Now()
	for i := 0; i < 500001; i++ {
		store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}
	writes := time.Since(start)

	start = time.Now()
	for i := 0; i < 50000; i++ {
		store.Get(fmt.Sprintf("key-%d", i))
	}
	readHits := time.Since(start)

	start = time.Now()
	for i := 0; i < 50000; i++ {
		store.Get(fmt.Sprintf("missing-%d", i))
	}
	readMisses := time.Since(start)

	return BenchmarkResult{useBloom, writes, readHits, readMisses}
}
