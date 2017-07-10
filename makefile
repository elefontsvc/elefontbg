# OUT = elefont
GOOS = windows
GOARCH = amd64

.PHONY: all build run

all: build

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build

run:
	./$(OUT)

install: build
	cp $(OUT) ../node_modules/.bin/$(OUT)
