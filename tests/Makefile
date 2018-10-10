DIFFS := $(shell ls *.expected | sed "s/\.expected//g")
BIN   := bin/

.PHONY: $(DIFFS)

all: run $(DIFFS)

run:
	./setup.sh
	./armq-tests

$(DIFFS):
	sed -i "s/$(VERSION)/master/g" $(BIN)$@
	diff -u $(BIN)$@ $@.expected
