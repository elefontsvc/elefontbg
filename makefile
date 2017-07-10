# OUT = elefont
GOOS = windows
GOARCH = amd64
BUILDVERSION = `date +%y%m%d%H%M%S`
LDFLAGS = -ldflags "-X main.buildVersion=$(BUILDVERSION)"

.PHONY: all build run

all: build

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) 

run:
	./$(OUT)

install: build
	cp $(OUT) ../node_modules/.bin/$(OUT)
