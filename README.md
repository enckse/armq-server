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

to run the server
```
armq_server
```

change the `rport` and `rserver` settings if redis is not co-located

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

```
systemctl start armqserver.service
```

## API

to execute the api, run the armq_api script
```
armq_api
```

### endpoints

to retrieve (non-data information from a bucket)
```
/armq/<bucket>/metadata
```

to retrieve the list of all buckets stored
```
/armq/buckets
```

to filter (`<after>` == epoch time)
```
/armq/buckets/<after>
```

to list the tags
```
/armq/tags
```

to filter (`<after>` == epoch time)
```
/armq/tags/<after>
```

to get tag data from a bucket (including next available bucket that _might_ have tag data)
```
/armq/tag/<tag>/data/<bucket>
```
