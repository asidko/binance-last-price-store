.PHONY: build test run stop status logs enable disable

DB_PATH ?= ./.data/ticks.db
SYMBOL := $(shell echo $(filter-out build test run stop status logs enable disable,$(MAKECMDGOALS)) | tr 'a-z' 'A-Z')

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

run:
	docker compose up --build -d

stop:
	docker compose down

status:
	@curl -s http://localhost:8080/status 2>/dev/null || printf "Binance Last Price Store\n\nStatus:     stopped (connection refused)\n"

logs:
	docker compose logs -f

enable:
	@mkdir -p $(dir $(DB_PATH))
	@sqlite3 $(DB_PATH) "CREATE TABLE IF NOT EXISTS symbol_settings (symbol TEXT PRIMARY KEY, enabled INTEGER DEFAULT 1);"
	@sqlite3 $(DB_PATH) "INSERT OR REPLACE INTO symbol_settings (symbol, enabled) VALUES ('$(SYMBOL)', 1);"
	@echo "Enabled $(SYMBOL)"

disable:
	@sqlite3 $(DB_PATH) "UPDATE symbol_settings SET enabled = 0 WHERE symbol = '$(SYMBOL)';"
	@echo "Disabled $(SYMBOL)"

%:
	@:
