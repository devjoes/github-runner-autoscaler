all: apiserver

apiserver: fmt vet
	go build -o bin/apiserver main.go

test: fmt vet
	go test ./... -coverprofile cover.out

run: fmt vet
	go run ./main.go

fmt:
	go fmt ./...

vet:
	go vet ./...