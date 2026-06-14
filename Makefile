# frp-panel build helpers.
# The Vue frontend (web/dist) is embedded into the Go binary, so the web build
# must run before `go build`.

BIN ?= frp-panel

.PHONY: build web go run dev clean

build: web go ## build frontend then compile the binary

web: ## build the Vue frontend into web/dist
	cd web && npm install && npm run build

go: ## compile the Go binary for the current platform
	go build -o $(BIN) .

run: build ## build then run
	./$(BIN)

dev: ## developer mode hint (run the two commands in separate terminals)
	@echo "Terminal 1:  go run . -addr :8088"
	@echo "Terminal 2:  cd web && npm run dev   (open http://localhost:5173)"

clean: ## remove build artifacts
	rm -f frp-panel frp-panel.exe
	rm -rf web/dist/* && touch web/dist/.gitkeep
