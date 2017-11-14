INSTALL=/usr/bin/
SRC=$(shell find . -type f | grep ".py$$")

all: analyze

install:
	cp -f armq_server.py $(INSTALL)armq_server
	cp -f armq_admin $(INSTALL)armq_admin
	cp -f armq_workers.py $(INSTALL)armq_workers
	cp -f service/armqserver.service /usr/lib/systemd/system/armqserver.service	

dependencies:
	pip install redis
	pip install git+https://github.com/systemd/python-systemd.git#egg=systemd

analyze:
	pip install pycodestyle pep257
	pycodestyle $(SRC)
	pep257 $(SRC)
