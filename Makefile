GO       ?= go
BIN       = $(CURDIR)/bin/primekit
BUILD     = $(GO) build -o $(BIN) ./cmd/primekit
DATA_FILES= primekit.bin primekit.db

.PHONY: build test clean reset-data reset-all

build:
	$(BUILD)

test:
	$(GO) test ./...

# Remove only known data files from CWD (safe — never uses rm -rf or globs)
reset-data:
	-rm -f $(DATA_FILES)

# Remove build output AND data files
reset-all: clean reset-data

clean:
	-rm -f $(BIN)
	$(GO) clean ./...

# Quick smoke test (rebuilds and runs basic checks without persisting)
smoke: build
	@$(BIN) isprime 17 | grep -q yes && echo "PASS isprime 17"
	@$(BIN) count 1000 | grep -q 168 && echo "PASS count 1000 = 168"
	@$(BIN) factor 1234567890 | grep -q "2 × 3 × 3 × 5 × 3607 × 3803" && echo "PASS factor 1234567890"
	@$(BIN) nth 100 | head -1 | grep -q 547 && echo "PASS nth 100 = 547"

# Full integration test (uses on-disk stores)
check: build reset-data
	$(BIN) sieve 1000
	$(BIN) isprime 997 | grep -q "stored"
	$(BIN) isprime 999 | grep -q "no"
	$(BIN) gaps 100
	$(BIN) bench
	$(BIN) status

bench: build
	$(BIN) bench

serve: build
	$(BIN) serve

.PHONY: build test clean reset-data reset-all smoke check bench serve
