TAG     :=
SERVER  := localhost
PORT    := 8080
TARGET  := http://$(SERVER):$(PORT)/
CURL    := curl -s $(TARGET)
TAGGED  := $(CURL)tags
API     := $(CURL)api
DATA    := $(CURL)?filter=fields.tag.raw:eq:$(TAG)\&limit=100000000
BIN     := bin/
JSON    := $(BIN)data.json

all: clean data outputs

tags:
	$(TAGGED)
	@echo

api:
	$(API)
	@echo

data:
ifndef TAG
	$(error no TAG given)
endif
	$(DATA) | python -c "import sys,json; print(json.dumps(json.loads(sys.stdin.read()), indent=4))" > $(JSON)

outputs:
	cat $(JSON) | python outputs.py $(TAG)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
