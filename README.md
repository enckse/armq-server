armq-server
===

In conjunction with [armq](https://voidedtech.com/cgit/armq/about/) - provides a receiving endpoint to extension payloads

# Description

armq-server works by using a reader over `/opt/armq/` or a socket receiver (for general TCP traffic from armq)

## Running

first you need to have deploy arma, adc, and armq (not documented here)

build
```
make
```

then install
```
make install
```

### armqserver

config file (default should work) is `/etc/armq.conf`, enable the following service to collect data
```
systemctl enable --now armqserver.service
```
**NOTE: armqserver MUST run with access to the directory armq writes to (e.g. `/opt/armq`)**

### armqapi

```
systemctl enable --now armqapi.service
```
**NOTE: armqapi MUST have access to the directory structure that armqserver uses as it's backend (this is NOT `/opt/armq` and is defined in the configuration file)**

to extract data via the api:
```
armq-extract
```

The ^ above ^ command will iterate through all tags (using the scanning rules in the configuration file)
* will find and output all tags and last time the tag was tracked
* download each tag data set

you can then run the following to briefly view a summary of the data
```
armq-summary <tag>.json
```
