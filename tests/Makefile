DIFFS := $(shell ls *.expected | sed "s/\.expected//g")

.PHONY: $(DIFFS)

all: run $(DIFFS)

run:
	./setup.sh
	./armq-tests

$(DIFFS):
	diff -u bin/$@ $@.expected
