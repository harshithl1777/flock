.PHONY: run test fmt

run:
	FLOCK_ENV=development go run ./cmd/flock

test:
	go test ./... -race

fmt:
	go fmt ./...
