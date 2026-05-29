package algo

func simpleSieve(limit uint64) []uint64 {
	if limit < 2 {
		return []uint64{}
	}
	isComposite := make([]bool, limit+1)
	primes := make([]uint64, 0, limit/10)
	for i := uint64(2); i <= limit; i++ {
		if !isComposite[i] {
			primes = append(primes, i)
			if i*i <= limit {
				for j := i * i; j <= limit; j += i {
					isComposite[j] = true
				}
			}
		}
	}
	return primes
}
