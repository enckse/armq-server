BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f -name "*.go" | grep -v "vendor/")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
APPS    := receiver api
COMMON  := $(CMD)common.go
GO      := go build $(FLAGS) -o $(BIN)armq-

all: clean server format

server: receiver api

receiver:
	$(GO)receiver $(CMD)receiver.go $(CMD)sockets.go $(CMD)files.go $(CMD)messages.go $(COMMON)

api:
	./converters.sh
	$(GO)api $(COMMON) $(CMD)api.go $(CMD)messages.go $(CMD)generated.go

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
