package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DaemonStore struct {
	baseURL string
	client  *http.Client
}

func NewDaemonStore(addr string) *DaemonStore {
	return &DaemonStore{
		baseURL: "http://" + addr,
		client:  &http.Client{},
	}
}

func (d *DaemonStore) Store(ctx context.Context, primes []uint64) error {
	if len(primes) == 0 {
		return nil
	}
	body, _ := json.Marshal(StoreRequest{Primes: primes})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/store", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("daemon request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("daemon: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon: %s: %s", resp.Status, string(b))
	}
	return nil
}

func (d *DaemonStore) Contains(ctx context.Context, n uint64) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/isprime/%d", d.baseURL, n), nil)
	if err != nil {
		return false, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var pr PrimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return false, err
	}
	return pr.IsPrime, nil
}

func (d *DaemonStore) Close() error {
	return nil
}
