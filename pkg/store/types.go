package store

type StoreRequest struct {
	Primes []uint64 `json:"primes"`
}

type PrimeResponse struct {
	Prime   uint64 `json:"prime,omitempty"`
	IsPrime bool   `json:"is_prime,omitempty"`
	Count   uint64 `json:"count,omitempty"`
	Error   string `json:"error,omitempty"`
}
