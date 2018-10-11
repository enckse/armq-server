BIN     := bin/
CMD     := src/
SRC     := $(shell find $(CMD) -type f -name "*.go")
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
GO      := go build $(FLAGS) -o $(BIN)armq-
APPS    := receiver api tests
GEN     := $(shell find . -type f -name "generated.go" | grep -v "vendor/")
API_GO  := $(CMD)api.go $(CMD)messages.go
COMMON  := $(CMD)generated.go $(CMD)common.go
TST_SRC := $(CMD)tests.go $(COMMON) $(API_GO)
MAIN    := $(CMD)main.go
API_SRC := $(COMMON) $(API_GO) $(MAIN)
RCV_SRC := $(COMMON) $(MAIN) $(CMD)sockets.go $(CMD)files.go $(CMD)receiver.go

.PHONY: $(APPS)

all: clean gen apps test format

apps: $(APPS)

receiver:
	$(GO)receiver $(RCV_SRC)

api:
	$(GO)api $(API_SRC)

tests:
	$(GO)tests $(TST_SRC)

gen:
	go generate $(CMD)setup.go

test: tests
	make -C tests VERSION=$(VERSION)

format:
	@echo $(SRC)
	exit $(shell goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -f $(GEN)
