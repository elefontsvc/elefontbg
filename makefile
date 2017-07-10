OUT = elefontbg.exe
# GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS)
GOOS = windows
GOARCH = amd64
ifeq ($(OS),Windows_NT)
	BUILDVERSION =`%TIME%`
else
	BUILDVERSION =`date +%y%m%d%H%M%S`
endif
# BUILDVERSION = `date +%y%m%d%H%M%S`
LDFLAGS = -ldflags "-X main.buildVersion=$(BUILDVERSION)"

.PHONY: all build run

all: build

build:
	go build $(LDFLAGS)

run:
	./$(OUT) debug

install: build
	cp $(OUT) ../node_modules/.bin/$(OUT)
