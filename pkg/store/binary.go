package store

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"

	"golang.org/x/exp/mmap"
)

const (
	magic    = 0x5052494D
	version  = uint32(1)
	headerSz = 24
)

type header struct {
	Magic    uint32
	Version  uint32
	Count    uint64
	MaxPrime uint64
}

type BinaryStore struct {
	path  string
	mu    sync.RWMutex
	file  *os.File
	data  []uint64
	count uint64
}

func NewBinaryStore(path string) (*BinaryStore, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	s := &BinaryStore{path: path, file: f}

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat store: %w", err)
	}

	if fi.Size() == 0 {
		h := header{Magic: magic, Version: version}
		if err := binary.Write(f, binary.LittleEndian, &h); err != nil {
			return nil, fmt.Errorf("write header: %w", err)
		}
		return s, nil
	}

	if fi.Size() < headerSz {
		return nil, errors.New("corrupt store: too small")
	}

	var h header
	if err := binary.Read(f, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if h.Magic != magic {
		return nil, fmt.Errorf("invalid magic: %x", h.Magic)
	}

	s.count = h.Count

	if h.Count > 0 {
		r, err := mmap.Open(path)
		if err != nil {
			return nil, fmt.Errorf("mmap store: %w", err)
		}
		data := make([]byte, r.Len())
		n, err := r.ReadAt(data, 0)
		r.Close()
		if err != nil {
			return nil, fmt.Errorf("read mmap: %w", err)
		}
		if n < headerSz {
			return nil, errors.New("corrupt store: header too short")
		}
		primeBytes := data[headerSz:]
		if uint64(len(primeBytes)) != h.Count*8 {
			return nil, fmt.Errorf("corrupt store: data size mismatch")
		}
		s.data = make([]uint64, h.Count)
		for i := uint64(0); i < h.Count; i++ {
			s.data[i] = binary.LittleEndian.Uint64(primeBytes[i*8 : (i+1)*8])
		}
	}

	return s, nil
}

func (s *BinaryStore) Store(ctx context.Context, primes []uint64) error {
	if len(primes) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !sort.SliceIsSorted(primes, func(i, j int) bool { return primes[i] < primes[j] }) {
		return errors.New("primes must be sorted")
	}

	if s.count > 0 && primes[0] <= s.data[s.count-1] {
		return errors.New("primes must be greater than stored max")
	}

	buf := make([]byte, len(primes)*8)
	for i, p := range primes {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], p)
	}

	if _, err := s.file.WriteAt(buf, int64(headerSz)+int64(s.count)*8); err != nil {
		return fmt.Errorf("write primes: %w", err)
	}

	s.count += uint64(len(primes))
	s.data = append(s.data, primes...)

	h := header{
		Magic:    magic,
		Version:  version,
		Count:    s.count,
		MaxPrime: primes[len(primes)-1],
	}
	if _, err := s.file.WriteAt(headerBytes(&h), 0); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	return nil
}

func headerBytes(h *header) []byte {
	buf := make([]byte, headerSz)
	binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
	binary.LittleEndian.PutUint32(buf[4:8], h.Version)
	binary.LittleEndian.PutUint64(buf[8:16], h.Count)
	binary.LittleEndian.PutUint64(buf[16:24], h.MaxPrime)
	return buf
}

func (s *BinaryStore) Contains(ctx context.Context, n uint64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return false, nil
	}
	idx := sort.Search(len(s.data), func(i int) bool { return s.data[i] >= n })
	return idx < len(s.data) && s.data[idx] == n, nil
}

func (s *BinaryStore) Count() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

func (s *BinaryStore) MaxPrime() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.count == 0 {
		return 0
	}
	return s.data[s.count-1]
}

func (s *BinaryStore) Data() []uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]uint64, len(s.data))
	copy(out, s.data)
	return out
}

func (s *BinaryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
