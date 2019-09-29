DIFFS := $(shell ls *.expected | sed "s/\.expected//g")
BIN   := bin/
DT    := $(shell date +%Y-%m-%d)
DS    := dataset/
SET   := bin/$(DT)/

.PHONY: $(DIFFS)

all: run $(DIFFS)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	mkdir -p $(SET)

run: clean
	for f in $(shell ls $(DS)); do cp $(DS)$$f $(SET).$(shell echo $$f | cut -d "." -f 2-); done
	go run ../tools/test.go

$(DIFFS):
	diff -u $(BIN)$@ $@.expected
