armq-server
===

In conjunction with [armq](https://github.com/enckse/armq) - provides a receiving endpoint to extension payloads

# Description

armq-server works by using a reader over `/dev/shm/armq/` or a socket receiver (for general TCP traffic from armq)

## Running

first you need to have deploy arma, adc, and armq (not documented here)

armq-server is available via the epiphyte [repository](https://mirror.epiphyte.network/repos/)
```
pacman -S armq-server
```

config file (default should work) is `/etc/armq.conf`, enable the following service to collect data
```
systemctl enable --now armqserver.service
```

and to enable the API
```
systemctl enable --now armqapi.service
```
