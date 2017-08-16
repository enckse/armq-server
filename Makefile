SRC=$(shell find . -type f | grep ".py$$")
.PHONY: all


install:
	pip install pep8 pep257

all: analyze

analyze:
	pep8 $(SRC)
	pep257 $(SRC)
