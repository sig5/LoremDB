package main

import (
	"fmt"
	"lorem-lsm/db"
	"time"
)

func main() {
	// store := db.NewLoremDB(true)

	// store.Put("name", "sakar")
	// store.Put("lang", "go")
	// store.Put("project", "lsm")

	// val, ok := store.Get("name")
	// fmt.Println(val, ok) // sakar true

	// store.Delete("lang")
	// val, ok = store.Get("lang")
	// fmt.Println(val, ok) // "" false

	// val, ok = store.Get("missing")
	// fmt.Println(val, ok) // "" false

	run_benchmark(true)
	run_benchmark(false)
}

func run_benchmark(useBloom bool) {
	fmt.Println("--- With bloom filter --- %s", useBloom)

	store := db.NewLoremDB(useBloom)

	// benchmark writes
	start := time.Now()
	for i := 0; i < 500001; i++ {
		store.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}
	fmt.Printf("10k writes: %v\n", time.Since(start))

	// benchmark reads (keys that exist)
	start = time.Now()
	for i := 0; i < 50000; i++ {
		store.Get(fmt.Sprintf("key-%d", i))
	}
	fmt.Printf("10k reads (hits): %v\n", time.Since(start))

	// benchmark reads (keys that don't exist)
	start = time.Now()
	for i := 0; i < 50000; i++ {
		store.Get(fmt.Sprintf("missing-%d", i))
	}
	fmt.Printf("10k reads (misses): %v\n", time.Since(start))

}
