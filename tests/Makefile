DIFFS := $(shell ls *.expected | sed "s/\.expected//g")
BIN   := bin/

.PHONY: $(DIFFS)

all: run $(DIFFS)

run:
	./setup.sh
	./armq-tests

$(DIFFS):
	diff -u $(BIN)$@ $@.expected
