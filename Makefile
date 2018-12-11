BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f) $(shell find common -type f)
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags 'i-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)'  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
GO      := go build $(FLAGS) -o $(BIN)armq-
APPS    := receiver api tests
GEN     := $(shell find . -type f -name "generated.go")

API_GO  := $(CMD)api.go $(CMD)messages.go
COMMON  := $(CMD)generated.go $(CMD)common.go
TST_SRC := $(CMD)tests.go $(COMMON) $(API_GO)
MAIN    := $(CMD)main.go
API_SRC := $(COMMON) $(API_GO) $(MAIN)
RCV_SRC := $(COMMON) $(MAIN) $(CMD)sockets.go $(CMD)files.go $(CMD)receiver.go

.PHONY: $(APPS)

all: clean gen apps test format

apps: $(APPS)

$(APPS):
	$(GO)$@ $(CMD)$@.go

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
