TAG     :=
SERVER  := localhost
PORT    := 8080
TARGET  := http://$(SERVER):$(PORT)/
CURL    := curl -s $(TARGET)
TAGGED  := $(CURL)tags
API     := $(CURL)api
DATA    := $(CURL)?filter=fields.tag.raw:eq:$(TAG)
BIN     := bin/
JSON    := $(BIN)data.json

all: clean tag outputs

tags:
	$(TAGGED)
	@echo

api:
	$(API)
	@echo

tag:
ifndef TAG
	$(error no TAG given)
endif
	$(DATA) > $(JSON)

outputs:
	cat $(JSON) | python outputs.py $(TAG)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
