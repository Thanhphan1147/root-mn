.PHONY: build test fmt vet web serve docker clean

build: ## compile everything
	go build ./...

test: ## run the test suite
	go test ./...

fmt: ## format Go sources
	gofmt -w .

vet: ## static analysis
	go vet ./...

web: ## rebuild docs/root.wasm + wasm_exec.js
	bash scripts/build-web.sh

serve: web ## rebuild the wasm and serve the static site locally
	go run ./cmd/rmn serve --addr :8080 --web docs

docker: ## build and run the container
	docker compose up -d --build

clean:
	rm -f rmn wan
