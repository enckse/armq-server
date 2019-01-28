BIN     := bin/
CMD     := cmd/
SRC     := $(shell find $(CMD) -type f) $(shell find internal/ -type f)
VERSION ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS   := -ldflags 'i-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)'  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
GO      := go build $(FLAGS) -o $(BIN)armq-
APPS    := receiver api tests
GEN     := $(shell find . -type f -name "generated.go")
SYSD    := /lib/systemd/system/

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

install:
	install -Dm 755 $(BIN)armq-receiver $(DESTDIR)/usr/bin/armq-receiver
	install -Dm 755 $(BIN)armq-api $(DESTDIR)/usr/bin/armq-api
	install -Dm 755 -d $(DESTDIR)$(SYSD)
	install -Dm 644 service/armqapi.service $(DESTDIR)$(SYSD)
	install -Dm 644 service/armqserver.service $(DESTDIR)$(SYSD)
