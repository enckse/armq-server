armq-server
===

In conjunction with [armq](https://github.com/enckse/armq) - provides a receiving endpoint to extension payloads


# Usage

to run the server
```
python armq_server.py
```

change the `rport` and `rserver` settings if redis is not co-located

# Admin

to save current state
```
python armq_server.py --mode admin --command flush
``` 

to stop
```
python armq_server.py --mode admin --command kill
``` 
