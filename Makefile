BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f -name "*.go" | grep -v "vendor/")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
APPS    := receiver api
COMMON  := $(CMD)common.go $(CMD)messages.go $(CMD)main.go
API     := $(CMD)api.go $(CMD)generated.go
GO      := go build $(FLAGS) -o $(BIN)armq-

all: clean server format

server: receiver api test

receiver:
	$(GO)receiver $(CMD)receiver.go $(CMD)sockets.go $(CMD)files.go $(COMMON)

api:
	./converters.sh
	$(GO)api $(COMMON) $(API)

test: api
	$(GO)test $(COMMON) $(CMD)harness.go 
	make -C tests FLAGS="$(FLAGS)"

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
