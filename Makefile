test:
	go test ./...
.PHONY: test

dist:
	mkdir -p dist

dist/gickup: dist
	go build -o dist/gickup ./main.go

build: dist/gickup
.PHONY: build

clean:
	$(RM) -r dist
.PHONY: clean

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.0
.PHONY: install-tools

lint:
	golangci-lint run
.PHONY: lint
