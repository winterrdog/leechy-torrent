#!/bin/bash

# build the release binaries for these architectures:
# - linux/amd64
# - linux/386
# - windows/amd64
# - windows/386
# - darwin/amd64
# - darwin/arm64

mkdir -p release/linux-amd64
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o release/linux-amd64/leechy-linux-amd64 main.go

mkdir -p release/linux-386
GOOS=linux GOARCH=386 go build -ldflags "-s -w" -o release/linux-386/leechy-linux-386 main.go

mkdir -p release/windows-amd64
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o release/windows-amd64/leechy-windows-amd64.exe main.go

mkdir -p release/windows-386
GOOS=windows GOARCH=386 go build -ldflags "-s -w" -o release/windows-386/leechy-windows-386.exe main.go

mkdir -p release/darwin-amd64
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o release/darwin-amd64/leechy-darwin-amd64 main.go

mkdir -p release/darwin-arm64
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o release/darwin-arm64/leechy-darwin-arm64 main.go