.PHONY: all build test lint

all: build

build:
	cd backend && go build -o ../bin/server ./...
	cd frontend && npm run build

test:
	cd backend && go test ./...
	cd frontend && npm test

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

proto:
	protoc --go_out=backend/pb --ts_out=frontend/src/pb proto/service.proto
