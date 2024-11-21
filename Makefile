help: Makefile
	@echo " Choose a command to run:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'

## start: Start service and all dependencies in Docker
start:
	docker compose up --build -d

## stop: Stop service and all dependencies
stop:
	docker compose down -v

## lint: Run linters
lint:
	golangci-lint run ./...

## test: Run unit tests
test:
	go test -race -v ./internal/... -count=1

## func-tests: Run func tests
func-tests:
	go test -race -v ./tests/... -count=1
