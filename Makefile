.PHONY: build clean test-integration build-cov coverage

OUT_DIR=./internal/utils/embedded-binaries
SRC=type-morph/main.ts

# Deno cross-compilation targets: deno_target:output_name
TARGETS = \
	x86_64-unknown-linux-gnu:type-morph-linux-amd64 \
	aarch64-unknown-linux-gnu:type-morph-linux-arm64 \
	x86_64-apple-darwin:type-morph-darwin-amd64 \
	aarch64-apple-darwin:type-morph-darwin-arm64 \
	x86_64-pc-windows-msvc:type-morph-windows-amd64.exe

build:
	@command -v deno >/dev/null 2>&1 || { \
		echo >&2 "❌ 'deno' is not installed. Please install it from https://deno.land/"; exit 1; \
	}
	@for pair in $(TARGETS); do \
		target=$${pair%%:*}; \
		outname=$${pair##*:}; \
		echo "Compiling type-morph for $$target → $$outname"; \
		deno compile \
			--allow-read \
			--allow-write \
			--allow-net \
			--allow-env \
			--target $$target \
			--output $(OUT_DIR)/$$outname \
			$(SRC); \
	done
	go run ./cmd/suprsend/main.go gendocs docs/
	go run ./cmd/suprsend/main.go genskills skills/

build-cov:
	go build -cover -o suprsend-cov ./cmd/suprsend/

test-integration: build-cov
	go test ./tests/integration/... -v -count=1

coverage: build-cov
	@mkdir -p /tmp/covdata
	@rm -f /tmp/covdata/*
	GOCOVERDIR=/tmp/covdata go test ./tests/integration/... -count=1
	go tool covdata textfmt -i=/tmp/covdata -o coverage.out
	go tool cover -html=coverage.out

clean:
	rm -f $(OUT_DIR)/type-morph-*
	rm -f suprsend-cov
