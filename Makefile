INSTALL=/usr/bin/
SRC=$(shell find . -type f | grep ".py$$")

all: analyze

install:
	install -Dm755 armq_server.py $(INSTALL)armq_server
	install -Dm755 armq_admin $(INSTALL)armq_admin
	install -Dm644 service/armqserver.service /usr/lib/systemd/system/armqserver.service
	install -Dm644 service/armqapi.service /usr/lib/systemd/system/armqapi.service
	install -Dm755 armq_api.py $(INSTALL)armq_api

dependencies:
	pip install redis
	pip install git+https://github.com/systemd/python-systemd.git#egg=systemd
	pip install flask

analyze:
	pip install pycodestyle pep257
	pycodestyle $(SRC)
	pep257 $(SRC)

endpoints:
	cat armq_api.py | grep "@app.route" | cut -d "(" -f 2 | cut -d ")" -f 1 | sed 's/"//g' | sort
