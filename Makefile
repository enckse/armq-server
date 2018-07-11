BIN          := bin/
SRC          := $(shell find . -type f -name "*.go" | grep -v "vendor/")
VERSION      := $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS        := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie

all: clean server format

server:
	go build -o $(BIN)armq-server $(FLAGS) $(MAIN) $(SRC)

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
