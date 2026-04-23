package bloom

import "hash/fnv"

type BloomFilter struct {
	bits []bool
	k    int
}

func NewBloomFilter(size int, k int) *BloomFilter {

	return &BloomFilter{
		k:    k,
		bits: make([]bool, size),
	}
}

func hash(key string, seed uint32) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	h.Write([]byte{byte(seed)})

	return h.Sum32()
}

func (bloomFilter *BloomFilter) Add(key string) {

	for i := 0; i < bloomFilter.k; i++ {
		hash := hash(key, uint32(i)) % (uint32(len(bloomFilter.bits)))
		bloomFilter.bits[hash] = true
	}
}

func (bloomFilter *BloomFilter) Contains(key string) bool {

	for i := 0; i < bloomFilter.k; i++ {
		if bloomFilter.bits[hash(key, uint32(i))%uint32(len(bloomFilter.bits))] == false {
			return false
		}
	}
	return true
}
