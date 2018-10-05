DIFFS := normal nohandlers limit skip

.PHONY: $(DIFFS)

all: run $(DIFFS)

run:
	./setup.sh
	./armq-test

$(DIFFS):
	diff -u bin/$@ $@.expected
