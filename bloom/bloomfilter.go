package bloom

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

// inline FNV-1a over string + seed byte — zero allocations
func fnv1a(key string, seed uint32) uint32 {
	h := uint32(2166136261) ^ seed
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return h
}

func (bloomFilter *BloomFilter) Add(key string) {
	for i := 0; i < bloomFilter.k; i++ {
		idx := fnv1a(key, uint32(i)) % uint32(len(bloomFilter.bits))
		bloomFilter.bits[idx] = true
	}
}

func (bloomFilter *BloomFilter) Contains(key string) bool {
	for i := 0; i < bloomFilter.k; i++ {
		if !bloomFilter.bits[fnv1a(key, uint32(i))%uint32(len(bloomFilter.bits))] {
			return false
		}
	}
	return true
}
