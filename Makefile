VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: run build build-all clean

run:
	go run ./cmd/server/

build:
	go build -ldflags="$(LDFLAGS)" -o course-helper.exe ./cmd/server/

build-all:
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o build/course-helper-windows-amd64.exe ./cmd/server/
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o build/course-helper-linux-amd64 ./cmd/server/
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o build/course-helper-darwin-amd64 ./cmd/server/

clean:
	rm -rf build/
	rm -f course-helper.exe course-helper
