SERVER  := 
PORT    := 8080
TARGET  := http://$(SERVER):$(PORT)/
CURL    := curl -s $(TARGET)
TAGGED  := $(CURL)tags
DATA    := $(CURL)?limit=100000000\&filter=fields.tag.raw:eq:
BIN     := bin/
TAGFILE := ".tag"
TAGS    := $(shell find . -name "*$(TAGFILE)")
TAGJ    := $(BIN)t.raw
PPRINT  := "import sys,json; print(json.dumps(json.loads(sys.stdin.read()), indent=4))"

.PHONY: $(TAGS)

all: clean data outputs

_check:
ifeq ($(SERVER),)
	$(error "make SERVER=server.domain required")
endif
	@echo

tags: _check
	$(TAGGED) | python -c $(PPRINT) > $(TAGJ)
	touch $(BIN)$(shell cat $(TAGJ) | python -c "import sys,json; print('{' + ','.join([' '.join(list(x.keys())) for x in json.loads(sys.stdin.read())['data']]) + '}')")$(TAGFILE)

$(TAGS): _check
	test -s $@.json || $(DATA)$(shell basename -s $(TAGFILE) $@) | python -c $(PPRINT) > $@.json
	cat $@.json | python outputs.py $(shell basename -s $(TAGFILE) $@)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
