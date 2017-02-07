# Exporting cAdvisor Stats to JSON via TCP/UDP (LogStash, etc.)

cAdvisor supports exporting stats as JSON over UDP or TCP. This can be used to send to a variety of services including Logstash. To use the JSON storage driver, you need to provide the additional flags to cAdvisor:

Set the storage driver as JSON:

```
 -storage_driver=json
```


Specify host address:

```
 -storage_driver_host="http://logstash:5000"
```

Specify protocol (supports "udp" and "tcp". Defaults to udp if flag is omitted)

```
-storage_driver_json_protocol="udp"
```

There is also an optional flag to add an additional info field to every json object sent. This can be useful to add any constant identifier:

```
 -storage_driver_json_info="some_identifier_or_other_string"
```


### Example for sending data to Logstash

To use with Logstash, add an input in logstash.conf to recieve the udp input and use the json codec:

```
udp {
    port => 5000
    codec => json
}
```

