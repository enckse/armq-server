VERSION ?= master
FLAGS   := -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)'  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
GEN_SRC := internal/common/generated.go
OBJECTS := armq-api armq-receiver

.PHONY: build test lint clean

build: $(OBJECTS) test lint

$(GEN_SRC): tools/setup.go
	go generate tools/setup.go

$(OBJECTS): $(GEN_SRC) $(shell find . -type f -name "*.go")
	go build $(FLAGS) -o $@ cmd/$@/main.go

test: tests
	make -C tests VERSION=$(VERSION)

lint:
	@golinter

clean:
	rm -f $(GEN_SRC)
	rm -f $(OBJECTS)
