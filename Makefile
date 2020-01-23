default: build

build:
	go build -o bin/edge cmd/edge/*
	go build -o bin/client cmd/client/*