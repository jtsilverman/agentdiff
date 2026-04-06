.PHONY: dev-api dev-frontend dev test build-api build-frontend

dev-api:
	cd api && go run . --port 8080

dev-frontend:
	cd frontend && npm run dev

dev:
	$(MAKE) dev-api & $(MAKE) dev-frontend

test:
	cd api && go test ./handlers/ -v

build-api:
	cd api && CGO_ENABLED=1 go build -o agentdiff-api .

build-frontend:
	cd frontend && npm run build
