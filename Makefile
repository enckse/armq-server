BIN     := bin/
VERSION ?= master
CMD     := cmd/
FLAGS   := -ldflags 'i-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)'  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
ARMQ    := $(BIN)armq-
GEN_SRC := internal/generated.go
STP_SRC := $(CMD)setup.go
FORMAT  := $(BIN)format
OBJECTS := $(ARMQ)api $(ARMQ)receiver $(ARMQ)tests

build: $(OBJECTS) test $(FORMAT)

$(GEN_SRC): $(STP_SRC)
	go generate $(STP_SRC)

$(OBJECTS): $(GEN_SRC) $(shell find . -type f -name "*.go")
	go build $(FLAGS) -o $@ $(CMD)$(shell basename $@ | cut -d "-" -f 2).go

test: tests
	make -C tests VERSION=$(VERSION)

$(FORMAT):
	@golinter
	@touch $(FORMAT)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -f $(GEN_SRC)
