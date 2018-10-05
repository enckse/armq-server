BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f -name "*.go" | grep -v "vendor/")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
COMMON  := $(CMD)common.go $(CMD)messages.go
API     := $(CMD)api.go $(CMD)generated.go
GO      := go build $(FLAGS) -o $(BIN)armq-
APPS    := receiver_app api_app test_app
GEND    := $(CMD)generated_

all: clean server format

server: $(APPS) receiver api test

$(APPS):
	./generate.sh

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
