SERVER  := localhost
PORT    := 8080
TARGET  := http://$(SERVER):$(PORT)/
CURL    := curl -s $(TARGET)
TAGGED  := $(CURL)tags
DATA    := $(CURL)?limit=100000000\&filter=fields.tag.raw:eq:
BIN     := bin/
TAGFILE := ".tag"
TAGS    := $(shell find . -name "*$(TAGFILE)") 

.PHONY: $(TAGS)

all: clean data outputs

tags:
	touch $(BIN)$(shell $(TAGGED) | python -c "import sys,json; print('{' + ','.join([' '.join(list(x.keys())) for x in json.loads(sys.stdin.read())['data']]) + '}')")$(TAGFILE)

$(TAGS):
	test -s $@.json || $(DATA)$(shell basename -s $(TAGFILE) $@) | python -c "import sys,json; print(json.dumps(json.loads(sys.stdin.read()), indent=4))" > $@.json
	cat $@.json | python outputs.py $(shell basename -s $(TAGFILE) $@)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
