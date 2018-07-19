# Exporting cAdvisor Stats to statsd

cAdvisor supports exporting stats to [statsd](https://github.com/etsy/statsd). To use statsd, you need to pass some additional flags to cAdvisor telling it where to find statsd:

Set the storage driver as statsd.

```
 -storage_driver=statsd
```

Specify what statsd instance to push data to:

```
 # The *ip:port* of the instance. Default is 'localhost:8086'
 -storage_driver_host=ip:port
```

# Examples

The easiest way to get up an running is to start the cadvisor binary with the `--storage_driver` and `--storage_driver_host` flags.

```
cadvisor --storage_driver="statsd" --storage_driver_host="localhost:8125"
```

The default port for statsd is 8125, so this wil start pumping metrics directly to it.