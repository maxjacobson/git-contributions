default: run

run:
    go run .

test:
    go test ./...

fmt:
    gofmt -w .

lint:
    go vet ./...
    test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)

build:
    go build -o bin/git-contributions .

install:
    go install .
