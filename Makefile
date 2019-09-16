BIN     := bin/
VERSION := $(BUILD_VERSION)
ifeq ($(VERSION),)
       VERSION := DEVELOP
endif
CMD     := cmd/
FLAGS   := -ldflags 'i-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)'  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -buildmode=pie
GOBUILD := go build $(FLAGS)
SYSD    := /lib/systemd/system/
ITL     := internal/
ARMQ    := $(BIN)armq-
ARMQRCV := $(ARMQ)receiver
ARMQAPI := $(ARMQ)api
ARMQTST := $(ARMQ)tests
RCV_SRC := $(CMD)receiver.go
ITERNL  := $(shell find $(ITL) -type f -name "*.go")
API_SRC := $(CMD)api.go
TST_SRC := $(CMD)tests.go
GEN_SRC := $(ITL)generated.go
STP_SRC := $(CMD)setup.go
FORMAT  := $(BIN)format

build: $(ARMQRCV) $(ARMQAPI) $(ARMQTST) test $(FORMAT)

$(GEN_SRC): $(STP_SRC)
	go generate $(STP_SRC)

$(ARMQRCV): $(GEN_SRC) $(RCV_SRC) $(ITERNL)
	$(GOBUILD) -o $(ARMQRCV) $(RCV_SRC)

$(ARMQAPI): $(GEN_SRC) $(API_SRC) $(ITERNL)
	$(GOBUILD) -o $(ARMQAPI) $(API_SRC)

$(ARMQTST): $(GEN_SRC) $(TST_SRC) $(ITERNL)
	$(GOBUILD) -o $(ARMQTST) $(TST_SRC)

test: tests
	make -C tests VERSION=$(VERSION)

$(FORMAT):
	goformatter
	@touch $(FORMAT)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -f $(GEN_SRC)

install:
	install -Dm 755 $(BIN)armq-receiver $(DESTDIR)/usr/bin/armq-receiver
	install -Dm 755 $(BIN)armq-api $(DESTDIR)/usr/bin/armq-api
	install -Dm 755 -d $(DESTDIR)$(SYSD)
	install -Dm 644 service/armqapi.service $(DESTDIR)$(SYSD)
	install -Dm 644 service/armqserver.service $(DESTDIR)$(SYSD)
