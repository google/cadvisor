# Exporting cAdvisor Stats to ElasticSearch

cAdvisor supports exporting stats to [ElasticSearch](https://www.elastic.co/). To use ES, you need to provide the
additional flags to cAdvisor:

Set the storage driver as ES:

```
 -storage_driver=elasticsearch
```

Specify ES host address:

```
 -storage_driver_es_host="http://elasticsearch:9200"
 # If you has several hosts, just use comma to separate it.
 -storage_driver_es_host="http://elasticsearch1:9200,http://elasticsearch2:9200,http://elasticsearch3:9200"
```

There are also optional flags:

```
 # ElasticSearch type name. By default it's "stats".
 -storage_driver_es_type="stats"
 # ElasticSearch can use a sniffing process to find all nodes of your cluster automatically. False by default.
 -storage_driver_es_enable_sniffer=false
 # ElasticSearch basic auth for http request, only works when any one of them is not empty
 -storage_driver_es_username="xxx"
 -storage_driver_es_password="xxx"
```

# Examples

For a detailed tutorial, see [docker-elk-cadvisor-dashboards](https://github.com/gregbkr/docker-elk-cadvisor-dashboards)
