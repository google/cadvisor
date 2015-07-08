# Exporting cAdvisor Stats to Riemann

cAdvisor supports sending stats to [Riemann](http://riemann.io). In order to set this up, you need to pass some additional configuration flags to cAdvisor.

Set the storage driver as Riemann:

```
 -storage_driver=riemann
```

Specify which Riemann instance to send data to:

```
 # The *ip:port* of the Riemann instance. Default is 'localhost:5555'
 -storage_driver_riemann_host=ip:port
```
 
Specify for how long the data is stored in a buffer before it gets flushed to Riemann:
 
```
 # The *duration* of the data stored in a buffer. Default is '1m'
 -storage_driver_buffer_duration=duration
```

Note, that for now cAdvisor supports only TCP transport for sending events to Riemann.
