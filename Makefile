VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)'  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
GEN_SRC := internal/generated.go
OBJECTS := armq-api armq-receiver armq-tests

.PHONY: build test lint clean

build: $(OBJECTS) test lint

$(GEN_SRC): cmd/setup.go
	go generate cmd/setup.go

$(OBJECTS): $(GEN_SRC) $(shell find . -type f -name "*.go")
	go build $(FLAGS) -o $@ cmd/$@.go

test: tests
	make -C tests VERSION=$(VERSION)

lint:
	@golinter

clean:
	rm -f $(GEN_SRC)
	rm -f $(OBJECTS)
