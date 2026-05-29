.PHONY: dev dev-api test build-api

dev-api:
	cd api && go run . --port 8080 --site ../site

dev: dev-api

test:
	cd api && go test ./handlers/ -v

build-api:
	cd api && CGO_ENABLED=1 go build -o agentdiff-api .
