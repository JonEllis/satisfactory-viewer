BUILDOPT := -ldflags '-s -w'
SOURCES  := $(wildcard *.go)
VERSION	 := $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)

build: fmt $(SOURCES) clean
	@$(foreach FILE, $(SOURCES), echo $(FILE); go build $(BUILDOPT) -o bin/`basename $(FILE) .go` $(FILE);)

fmt:
	@$(foreach FILE, $(SOURCES), go fmt $(FILE);)

clean:
	rm -f bin/*
