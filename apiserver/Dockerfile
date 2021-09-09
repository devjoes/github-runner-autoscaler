FROM alpine:latest  as cacerts
RUN apk --no-cache add ca-certificates

# Build the apiserver binary
FROM golang:1.15 as builder
RUN apt update && apt install memcached -y && apt clean && service memcached start
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
RUN go get github.com/devjoes/github-runner-autoscaler/operator@add-forced-scale-up

# Copy the go source
COPY main.go main.go
COPY pkg/ pkg/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o apiserver main.go
RUN go test ./...

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/apiserver .
COPY --from=cacerts /etc/ca-certificates/ /etc/ca-certificates/
USER 65532:65532

ENTRYPOINT ["/apiserver"]