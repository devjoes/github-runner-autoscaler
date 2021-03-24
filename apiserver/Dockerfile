FROM golang:1.16 AS builder
WORKDIR /go/src/github.com/devjoes/runner-scaler/
COPY operator /go/src/github.com/devjoes/operator/
COPY apiserver/go.* .
RUN go mod download
COPY apiserver/ .


# COPY go.* .
# RUN go mod download
# COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
RUN go test ./...

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/devjoes/runner-scaler/app .
ENTRYPOINT [ "./app" ]