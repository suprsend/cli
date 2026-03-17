.PHONY: build clean

OUT=./internal/utils/embedded-binaries/type-morph
SRC=type-morph/main.ts

build:
	@command -v deno >/dev/null 2>&1 || { \
		echo >&2 "❌ 'deno' is not installed. Please install it from https://deno.land/"; exit 1; \
	}
	deno compile \
		--allow-read \
		--allow-write \
		--allow-net \
		--allow-env \
		--output $(OUT) \
		$(SRC)
	go run ./cmd/suprsend/main.go gendocs docs/
	go run ./cmd/suprsend/main.go genskills skills/
clean:
	rm -f $(OUT)