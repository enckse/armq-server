BIN     := bin/
CMD     := src/
SRC     := $(shell find . -type f -name "*.go" | grep -v "vendor/")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
GO      := go build $(FLAGS) -o $(BIN)armq-
GEN     := $(CMD)generated
APPS    := receiver api

.PHONY: $(APPS)

all: clean server format

server: gen $(APPS) test

gen:
	go generate $(CMD)setup.go

$(APPS):
	$(GO)$@ $@/*.go

test: api
	$(GO)test $(COMMON) $(API) $(CMD)harness.go
	make -C tests FLAGS="$(FLAGS)"

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -f $(GEN)*
