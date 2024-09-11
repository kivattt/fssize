#!/usr/bin/env bash
bin=./bin

if [ ! -d $bin ]; then
	mkdir $bin
fi

ldflags="-s -w"

# amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fssize-linux-amd64

# i386
GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fssize-linux-i386

# arm64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fssize-linux-arm64

# arm
GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fssize-linux-arm
