.PHONY: all


install:
	pip install pep8 pep257

all: analyze

analyze:
	pep8 *.py
	pep257 *.py
