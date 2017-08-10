armq-server
===

In conjunction with [armq](https://github.com/enckse/armq) - provides a receiving endpoint to extension payloads

# Description

armq-server works by using ZMQ as a STREAM receiver (for general TCP traffic from armq). It takes this data, background saves to redis, and immediately acknowledges the client with an `ack` response

# Usage

to run the server
```
python armq_server.py
```

change the `rport` and `rserver` settings if redis is not co-located

[![Build Status](https://travis-ci.org/enckse/armq-server.svg?branch=master)](https://travis-ci.org/enckse/armq-server)

# Admin

to save current state
```
python armq_server.py --mode admin --command flush
``` 

to stop
```
python armq_server.py --mode admin --command kill
``` 
