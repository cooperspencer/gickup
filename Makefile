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
