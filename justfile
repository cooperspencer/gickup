lasttag := ```
	git describe --tags `git rev-list --tags --max-count=1`
	```

build:
	go build .

release:
	git checkout {{lasttag}}
	go build -ldflags '-X "main.version={{lasttag}}"'
	
cleanup:
	go mod tidy
	gofmt -s -w .
	
update:
	go get -u

pull:
	git pull
