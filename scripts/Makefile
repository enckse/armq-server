SERVER  := 
PORT    := 8080
TARGET  := http://$(SERVER):$(PORT)/
CURL    := curl -s $(TARGET)
TAGGED  := $(CURL)tags
DATA    := $(CURL)?limit=0\&filter=fields.tag.raw:eq:
BIN     := bin/
TAGFILE := ".tag"
TAGS    := $(shell find . -name "*$(TAGFILE)")
TAGJ    := $(BIN)t.raw
PPRINT  := "import sys,json; print(json.dumps(json.loads(sys.stdin.read()), indent=4))"

.PHONY: $(TAGS)

all: clean tags

_check:
ifeq ($(SERVER),)
	$(error "make SERVER=server.domain required")
endif
	@echo $(SERVER) selected...

_tags:
	$(TAGGED) | python -c $(PPRINT) > $(TAGJ)

tags: _check _tags list

list:
	@echo
	@for f in $(shell cat $(TAGJ) | grep -E '          (\"[a-z]|    \"20)' | sed "s/ //g;s/:\[//g;s/\"//g" | tr '\n' ' ' | sed 's/ 2/_2/g' | tr ' ' '\n' | sed "s/[_]/ /g" | sort -k 2); do \
		echo $$f; \
		touch $(BIN)$$f$(TAGFILE); \
		done
	@rm -f $(BIN)*-*
	@echo

$(TAGS): _check
	test -s $@.json || $(DATA)$(shell basename -s $(TAGFILE) $@) | python -c $(PPRINT) > $@.json
	cat $@.json | python outputs.py $(shell basename -s $(TAGFILE) $@)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
