INSTALL=/usr/bin/
SRC=$(shell find . -type f | grep ".py$$")

all: analyze

install:
	cp -f armq_server.py $(INSTALL)armq_server
	cp -f armq_admin $(INSTALL)armq_admin
	cp -f armq_workers.py $(INSTALL)armq_workers
	cp -f service/armqserver.service /usr/lib/systemd/system/armqserver.service	

analyze:
	pip install pep8 pep257
	pep8 $(SRC)
	pep257 $(SRC)
