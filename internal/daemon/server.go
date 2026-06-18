package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ishaandevburman/primekit/pkg/algo"
	"github.com/ishaandevburman/primekit/pkg/store"
)

type Server struct {
	binStore *store.BinaryStore
	sqStore  *store.SQLiteStore
	gen      *algo.SegmentedSieve
	server   *http.Server
}

type StatusResponse struct {
	PrimesStored uint64 `json:"primes_stored"`
	MaxPrime     uint64 `json:"max_prime"`
	Segments     int    `json:"segments"`
	Uptime       string `json:"uptime"`
}

func NewServer(binPath, dbPath string) (*Server, error) {
	bs, err := store.NewBinaryStore(binPath)
	if err != nil {
		return nil, fmt.Errorf("binary store: %w", err)
	}
	sq, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		bs.Close()
		return nil, fmt.Errorf("sqlite store: %w", err)
	}
	return &Server{
		binStore: bs,
		sqStore:  sq,
		gen:      algo.NewSegmentedSieve(1 << 20),
		server:   &http.Server{},
	}, nil
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/store", s.handleStore)
	mux.HandleFunc("/isprime/", s.handleIsPrime)
	mux.HandleFunc("/nth/", s.handleNth)
	mux.HandleFunc("/primes", s.handlePrimes)
	mux.HandleFunc("/status", s.handleStatus)

	s.server = &http.Server{
		Handler:      withCORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	log.Printf("primekit daemon listening on %s", addr)
	return s.server.Serve(ln)
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		s.binStore.Close()
		s.sqStore.Close()
		return err
	}
	s.binStore.Close()
	return s.sqStore.Close()
}

func (s *Server) handleStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req store.StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, store.PrimeResponse{Error: err.Error()})
		return
	}
	if err := s.binStore.Store(r.Context(), req.Primes); err != nil {
		writeJSON(w, http.StatusInternalServerError, store.PrimeResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, store.PrimeResponse{Count: uint64(len(req.Primes))})
}

func (s *Server) handleIsPrime(w http.ResponseWriter, r *http.Request) {
	n, err := extractUint(r.URL.Path, "/isprime/")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, store.PrimeResponse{Error: err.Error()})
		return
	}
	found, _ := s.binStore.Contains(r.Context(), n)
	if found {
		writeJSON(w, http.StatusOK, store.PrimeResponse{Prime: n, IsPrime: true})
		return
	}
	mr := &algo.MillerRabin{}
	writeJSON(w, http.StatusOK, store.PrimeResponse{Prime: n, IsPrime: mr.IsPrime(r.Context(), n)})
}

func (s *Server) handleNth(w http.ResponseWriter, r *http.Request) {
	n, err := extractUint(r.URL.Path, "/nth/")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, store.PrimeResponse{Error: err.Error()})
		return
	}
	if s.binStore.Count() > n {
		primes := s.binStore.Data()
		writeJSON(w, http.StatusOK, store.PrimeResponse{Prime: primes[n]})
		return
	}
	p, err := s.gen.NthPrime(r.Context(), n)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, store.PrimeResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, store.PrimeResponse{Prime: p})
}

func (s *Server) handlePrimes(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		writeJSON(w, http.StatusBadRequest, store.PrimeResponse{Error: "missing limit parameter"})
		return
	}
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, store.PrimeResponse{Error: err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"primes":[`))
	first := true
	ch := make(chan uint64, 4096)
	go func() {
		s.gen.Primes(r.Context(), limit, ch)
	}()
	for p := range ch {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte(strconv.FormatUint(p, 10)))
		first = false
	}
	w.Write([]byte(`]}`))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	segs, _ := s.sqStore.ListSegments(r.Context())
	writeJSON(w, http.StatusOK, StatusResponse{
		PrimesStored: s.binStore.Count(),
		MaxPrime:     s.binStore.MaxPrime(),
		Segments:     len(segs),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func extractUint(path, prefix string) (uint64, error) {
	s := path[len(prefix):]
	if s == "" {
		return 0, fmt.Errorf("missing value")
	}
	return strconv.ParseUint(s, 10, 64)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
