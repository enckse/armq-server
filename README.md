armq-server
===

In conjunction with [armq](https://github.com/enckse/armq) - provides a receiving endpoint to extension payloads

# Description

armq-server works by using a socket receiver (for general TCP traffic from armq). It takes this data and background saves to redis

# Install

clone this repository and navigate to it:
```
make install
```

# Usage

to run the server to collect data
```
armq_server
```


[![Build Status](https://travis-ci.org/enckse/armq-server.svg?branch=master)](https://travis-ci.org/enckse/armq-server)

# Admin

to save current state
```
armq_admin flush
``` 

to stop
```
armq_admin kill
``` 

# Service

to enable the receiving endpoint
```
systemctl start armqserver.service
```

to enable the web API to query data received
```
systemctl start armqapi.service
```

## API

to execute the api, run the armq_api script
```
armq_api
```

navigate to server root (e.g. "http://localhost:port/" for documentation)
