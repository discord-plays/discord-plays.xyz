all:
	make build-main

build-main:
	mkdir -p dist/ && go build -o dist/discord-plays-xyz .

dev:
	mkdir -p dist/
	go build -tags debug -o dist/discord-plays-xyz .
	DEV_FOLDER="$(pwd)" ./dist/discord-plays-xyz
