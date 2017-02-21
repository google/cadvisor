# Exporting cAdvisor Stats to ElasticSearch

cAdvisor supports exporting stats to [ElasticSearch](https://www.elastic.co/). To use ES, you need to provide the additional flags to cAdvisor:

Set the storage driver as ES:

```
 -storage_driver=elasticsearch
```

Specify ES host address:

```
 -storage_driver_es_host="http://elasticsearch:9200"
```

There are also optional flags:

```
 # ElasticSearch type name. By default it's "stats".
 -storage_driver_es_type="stats"
 # ElasticSearch can use a sniffing process to find all nodes of your cluster automatically. False by default.
 -storage_driver_es_enable_sniffer=false
```

## ElasticSearch 5.x support
Set the storage driver as ES:

```
 -storage_driver=elasticsearch.v5
```
There are also optional flags:

* storage_driver_es_basic_auth 
* storage_driver_es_sniffer_timeout
* storage_driver_es_sniffer_timeout_startup
* storage_driver_es_sniffer_interval
* storage_driver_es_enable_health_check
* storage_driver_es_health_check_timeout
* storage_driver_es_health_check_timeout_startup
* storage_driver_es_health_check_interval

all options explaination can be found at  project [olivere/elastic](https://github.com/olivere/elastic/wiki).

### index name time support
Enable golang time format support for option **storage_driver_es_index**.

#### Example
```
  -storage_driver_es_index="cadvisor-{2006.01.02}"
```
Data will be exported to index cadvisor-2017.01.01 on day 2017/01/01, index cadvisor-2017.02.12 on day 2017.02.12.


# Examples

For a detailed tutorial, see [docker-elk-cadvisor-dashboards](https://github.com/gregbkr/docker-elk-cadvisor-dashboards)
