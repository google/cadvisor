# Exporting cAdvisor Stats to InfluxDB

cAdvisor supports exporting stats to [InfluxDB](https://www.influxdata.com/) 1.x, 2.x, InfluxDB 3 Core and InfluxDB Cloud Serverless.

Set the storage driver as InfluxDB.

```
 -storage_driver=influxdb
```

## InfluxDB 1.x

Specify what InfluxDB instance to push data to:

```
 # The *ip:port* of the database. Default is 'localhost:8086'
 -storage_driver_host=ip:port
 # database name. Uses db 'cadvisor' by default
 -storage_driver_db
 # database username. Default is 'root'
 -storage_driver_user
 # database password. Default is 'root'
 -storage_driver_password
 # Use secure connection with database. False by default
 -storage_driver_secure
 # Writes will be buffered for this duration, and committed to the non memory backends as a single transaction. Default is '60s'
 -storage_driver_buffer_duration
 # retention policy. Default is '' which corresponds to the default retention policy of the influxdb database
-storage_driver_influxdb_retention_policy
```

## InfluxDB 2.x, InfluxDB 3 Core and InfluxDB Cloud

InfluxDB 2.x and later do not use usernames and passwords for API access; they use API tokens instead. To write through the InfluxDB v2 API (which is also implemented by InfluxDB 3 Core), set `-storage_driver_influxdb_auth_token`. The `-storage_driver_user` and `-storage_driver_password` flags are then ignored.

```
 # The *ip:port* of the database. Default is 'localhost:8086' (InfluxDB 3 Core defaults to 'localhost:8181')
 -storage_driver_host=ip:port
 # database name. Used as the InfluxDB 2.x bucket (or the InfluxDB 3 Core database) unless -storage_driver_influxdb_bucket is set. Uses db 'cadvisor' by default
 -storage_driver_db
 # InfluxDB API token. Required to enable token-based writes.
 -storage_driver_influxdb_auth_token
 # InfluxDB organization. Required by InfluxDB 2.x; ignored by InfluxDB 3 Core.
 -storage_driver_influxdb_org
 # InfluxDB 2.x bucket or InfluxDB 3 Core database. Defaults to the -storage_driver_db value.
 -storage_driver_influxdb_bucket
 # Use secure connection with database. False by default
 -storage_driver_secure
 # Writes will be buffered for this duration, and committed to the non memory backends as a single transaction. Default is '60s'
 -storage_driver_buffer_duration
```

Notes:

- The bucket (or InfluxDB 3 Core database) must exist before starting cAdvisor. Create it with `influx bucket create` (InfluxDB 2.x) or `influxdb3 create database` (InfluxDB 3 Core).
- The retention policy flag does not apply in token mode; retention is a property of the InfluxDB 2.x bucket.
- InfluxDB 3 Core ignores the organization, but the InfluxDB 2.x API requires the query parameter, so set `-storage_driver_influxdb_org` to any non-empty string.

### Example: cAdvisor with InfluxDB 2.x

```console
$ influx bucket create --name cadvisor --org my-org --token $INFLUX_TOKEN
$ docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:ro \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --volume=/dev/disk/:/dev/disk:ro \
  --publish=8080:8080 \
  --detach=true \
  --name=cadvisor \
  gcr.io/cadvisor/cadvisor:vlatest \
    -storage_driver=influxdb \
    -storage_driver_host=influxdb:8086 \
    -storage_driver_db=cadvisor \
    -storage_driver_influxdb_auth_token=$INFLUX_TOKEN \
    -storage_driver_influxdb_org=my-org
```

### Example: cAdvisor with InfluxDB 3 Core

```console
$ influxdb3 create database --database cadvisor
$ influxdb3 create token --admin  # prints the admin token
$ docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:ro \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --volume=/dev/disk/:/dev/disk:ro \
  --publish=8080:8080 \
  --detach=true \
  --name=cadvisor \
  gcr.io/cadvisor/cadvisor:vlatest \
    -storage_driver=influxdb \
    -storage_driver_host=influxdb3:8181 \
    -storage_driver_db=cadvisor \
    -storage_driver_influxdb_auth_token=$INFLUX3_TOKEN \
    -storage_driver_influxdb_org=cadvisor
```

# Examples

[Brian Christner](https://www.brianchristner.io) wrote a detailed post on [setting up Docker monitoring](https://www.brianchristner.io/how-to-setup-docker-monitoring) with cAdvisor and Influxdb.  A docker compose configuration for setting up cadvisor-influxdb-grafana can be found [here](https://github.com/dalekurt/docker-monitoring/blob/master/docker-compose.yml).
