# Exporting cAdvisor Stats to Splunk Enterprise or Splunk Cloud instance

cAdvisor supports exporting stats to [Splunk Enterprise or Splunk Cloud instance](https://splunk.com). To use Splunk Enterprise or Splunk Cloud instance, you need to pass some additional flags to cAdvisor telling it where the Splunk instance is located:

Set the storage driver as Splunk.

```
 -storage_driver=splunk
```

Specify what Splunk instance to push data to:

```
 # The *protocol://ip:port* of the Splunk Http Event Collector endpoint. Default is 'https://localhost:8088'
 -storage_driver_splunk_url=https://ip:port
 # Splunk token for authorization. Uses '00000000-0000-0000-0000-000000000000' by default
 -storage_driver_splunk_token
 # Splunk event source.
 -storage_driver_splunk_source
 # Splunk event source type.
 -storage_driver_splunk_source_type
 # Splunk event index.
 -storage_driver_splunk_index
 # Path to root certificate.
 -storage_driver_splunk_capath
 # Name to use for validating server certificate; by default the hostname of the `splunk-url` will be used.
 -storage_driver_splunk_caname
 # Ignore server certificate validation.
 -storage_driver_splunk_insecureskipverify
 # Verify on start, that cAdvisor can connect to Splunk server.
 -storage_driver_splunk_verifyconnection
 # Enable gzip compression.
 -storage_driver_splunk_gzip
 # Set gzip compression level.
 -storage_driver_splunk_gzip_level
```

# Examples

Start cadvisor inside of container with Splunk as storage.

```
./cadvisor -storage_driver splunk -storage_driver_splunk_url https://2b0672df443d:8088 -storage_driver_splunk_token DBB64462-5A16-4497-A40F-CE0C10FF85D4
```

Or start official cAdvisor image with Splunk as storage driver.

```
docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --publish=8080:8080 \
  --detach=true \
  --name=cadvisor \
  google/cadvisor:latest \
  cadvisor -storage_driver splunk -storage_driver_splunk_url https://2b0672df443d:8088 -storage_driver_splunk_token DBB64462-5A16-4497-A40F-CE0C10FF85D4
```

## Advanced options

Splunk Storage allows you to configure few advanced options by specifying next environment variables for the Docker daemon.

| Environment variable name                 | Default value | Description                                                                                                                                        |
|-------------------------------------------|---------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| `SPLUNK_STORAGE_POST_MESSAGES_FREQUENCY`  | `5s`          | If there is nothing to batch how often driver will post messages. You can think about this as maximum time of waiting more more messages to batch. |
| `SPLUNK_STORAGE_POST_MESSAGES_BATCH_SIZE` | `1000`        | How many messages driver should wait before sending them in one batch.                                                                             |
| `SPLUNK_STORAGE_BUFFER_MAX`               | `10 * 1000`   | If driver cannot connect to remote server, what is the maximum amount of messages it can hold in buffer for retries.                               |
| `SPLUNK_STORAGE_CHANNEL_SIZE`             | `4 * 1000`    | How many pending messages can be in the channel which is used to send messages to background logger worker, which batches them.                    |
