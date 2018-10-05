DIFFS := normal nohandlers

all: run $(DIFFS)

run:
	./setup.sh
	./armq-test

$(DIFFS):
	diff -u bin/$@ $@.expected
