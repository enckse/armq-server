BIN     := bin/
CMD     := src/
SRC     := $(shell find . -type f -name "*.go" | grep -v "vendor/")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
GO      := go build $(FLAGS) -o $(BIN)armq-
GEN     := $(CMD)generated
API     := api
TEST    := tests
DISTRIB := $(API) $(TEST)
APPS    := receiver $(API) $(TEST)

.PHONY: $(APPS)

all: clean server format

server: gen $(DISTRIB) $(APPS) test

gen:
	go generate $(CMD)setup.go

$(APPS):
	cp $(CMD)generated.go $@/
	$(GO)$@ $@/*.go

test: $(TEST)
	make -C tests FLAGS="$(FLAGS)"

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -f $(GEN)*
