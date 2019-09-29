DIFFS := $(shell ls *.expected | sed "s/\.expected//g")
BIN   := bin/

.PHONY: $(DIFFS)

all: run $(DIFFS)

run:
	./setup.sh
	go run ../tools/test.go

$(DIFFS):
	diff -u $(BIN)$@ $@.expected
