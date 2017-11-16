INSTALL=/usr/bin/
SRC=$(shell find . -type f | grep ".py$$")

all: analyze

install:
	install -Dm755 armq_server.py $(INSTALL)armq_server
	install -Dm755 armq_admin $(INSTALL)armq_admin
	install -Dm644 service/armqserver.service /usr/lib/systemd/system/armqserver.service	

dependencies:
	pip install redis
	pip install git+https://github.com/systemd/python-systemd.git#egg=systemd

analyze:
	pip install pycodestyle pep257
	pycodestyle $(SRC)
	pep257 $(SRC)
