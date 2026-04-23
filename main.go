package main

import (
	"fmt"
	"lorem-lsm/bloom"
)

func main() {
	testFilter := bloom.NewBloomFilter(1000, 5)
	testFilter.Add("test")
	fmt.Println(testFilter.Contains("test"))
	fmt.Println(testFilter)
	// should be false, but might return true (false positive)
	fmt.Println(testFilter.Contains("gopher"))
	fmt.Println(testFilter.Contains("rust"))
	fmt.Println(testFilter.Contains("lsm"))

}
