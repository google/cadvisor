# Exporting cAdvisor Stats to statsd

cAdvisor supports exporting stats to [statsd](https://github.com/etsy/statsd). To use statsd, you need to pass some additional flags to cAdvisor telling it where to find statsd:

Set the storage driver as statsd.

```
 -storage_driver=statsd
```

Specify what statsd instance to push data to:

```
 # The *ip:port* of the database. Default is 'localhost:8086'
 -storage_driver_host=ip:port
```

# Examples