all:
	make build-main

build-main:
	mkdir -p dist/ && go build -o dist/discord-plays-xyz .
