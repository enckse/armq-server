BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f -name "*.go" | grep -v "vendor/")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
COMMON  := $(CMD)common.go $(CMD)messages.go
API     := $(CMD)api.go $(CMD)generated.go
GO      := go build $(FLAGS) -o $(BIN)armq-
GEN     := $(CMD)generated
GEND    := $(GEN)_

all: clean server format

server: gen receiver api test

gen:
	go generate $(CMD)setup.go

receiver:
	$(GO)receiver $(GEND)receiver.go $(CMD)receiver.go $(CMD)sockets.go $(CMD)files.go $(COMMON)

api:
	$(GO)api $(COMMON) $(API) $(GEND)api.go

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
