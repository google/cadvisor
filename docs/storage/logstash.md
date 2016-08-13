# Exporting cAdvisor Stats to LogStash

cAdvisor supports exporting stats to [LogStash](https://www.elastic.co/products/logstash/). The stats will be sent as JSON across UDP. To use LogStash, you need to provide the additional flags to cAdvisor:

Set the storage driver as Logstash:

```
 -storage_driver=logstash
```


Specify LogStash host address:

```
 -storage_driver_host="http://logstash:5000"
```

There is also an optional flag to set the type:

```
 # Logstash type name. By default it is "cadvisor".
 -storage_driver_logstash_type="stats"
```

Example input in logstash.conf:

```
udp {
    port => 5000
    codec => json
}
```

