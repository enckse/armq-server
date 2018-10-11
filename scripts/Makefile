TAG     :=
SERVER  := localhost
PORT    := 8080
TARGET  := http://$(SERVER):$(PORT)/
DATA    := curl -s $(TARGET)?filter=fields.tag.raw:eq:$(TAG)
BIN     := bin/
JSON    := $(BIN)data.json

all: clean tag outputs

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
