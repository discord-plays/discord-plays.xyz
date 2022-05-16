.PHONY: build dev

build:
	mkdir -p dist/ && go build -o dist/discord-plays-xyz ./cmd/discord-plays-xyz

dev:
	mkdir -p dist/
	go build -tags debug -o dist/discord-plays-xyz ./cmd/discord-plays-xyz
	./dist/discord-plays-xyz
