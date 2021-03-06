armq-server
===

In conjunction with [armq](https://github.com/enckse/armq) - provides a receiving endpoint to extension payloads

armq-server works by being a reader over `/opt/armq/` and using workers to process and archive data

## running

first you need to have deployed arma, adc, and armq (not documented here)

### build

```
make
```

### armqserver

config file (default should work) is `/etc/armq.conf`, enable the following service to collect data
```
systemctl enable --now armqserver.service
```

**NOTE: armqserver MUST run with access to the directory armq writes to (e.g. `/opt/armq`)**

to extract data:
```
armq-api
```

**NOTE: `armq-api` MUST have access to the directory structure that armqserver uses as it's backend (this is NOT `/opt/armq` and is defined in the configuration file)**

The ^ above ^ command will iterate through all tags (using the scanning rules in the configuration file)

* will find and output all tags and last time the tag was tracked
* download each tag data set
