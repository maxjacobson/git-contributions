default: run

run:
    go run .

test:
    go test ./...

build:
    go build -o bin/git-contributions .

install:
    go install .
