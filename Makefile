BINARY   := batch-cost-go
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: build run test vet lint clean install \
        release snapshot \
        llm-download llm-start llm-stop llm-status

# ── Build ─────────────────────────────────────────────────────────────────────

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf dist/

# ── Dev ───────────────────────────────────────────────────────────────────────

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint nicht installiert — nur go vet"

# ── Release ───────────────────────────────────────────────────────────────────

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

# ── LLM ───────────────────────────────────────────────────────────────────────

llm-download: build
	./$(BINARY) llm download

llm-start: build
	./$(BINARY) llm start

llm-stop: build
	./$(BINARY) llm stop

llm-status: build
	./$(BINARY) llm status
