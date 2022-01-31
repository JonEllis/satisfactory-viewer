VERSION	 := $(shell git describe --tags --always | sed 's/-/+/' | sed 's/^v//')
BUILDOPT := -ldflags "-s -w -X main.version=$(VERSION)"
SOURCES  := $(wildcard *.go)

build: fmt $(SOURCES) clean
	@$(foreach FILE, $(SOURCES), echo $(FILE); go build $(BUILDOPT) -o bin/`basename $(FILE) .go` $(FILE);)

fmt:
	@$(foreach FILE, $(SOURCES), go fmt $(FILE);)

clean:
	rm -f bin/*
