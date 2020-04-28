run: build
	heroku local

build:
	go build -o ./bin/ynn -v .
