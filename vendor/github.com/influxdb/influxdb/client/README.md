# InfluxDB Client

[![GoDoc](https://godoc.org/github.com/influxdb/influxdb?status.svg)](http://godoc.org/github.com/influxdb/influxdb/client/v2)

## Description

**NOTE:** The Go client library now has a "v2" version, with the old version
being deprecated. The new version can be imported at
`import "github.com/influxdb/influxdb/client/v2"`. It is not backwards-compatible.

A Go client library written and maintained by the **InfluxDB** team.
This package provides convenience functions to read and write time series data.
It uses the HTTP protocol to communicate with your **InfluxDB** cluster.


## Getting Started

### Connecting To Your Database

Connecting to an **InfluxDB** database is straightforward. You will need a host
name, a port and the cluster user credentials if applicable. The default port is
8086. You can customize these settings to your specific installation via the
**InfluxDB** configuration file.

Thought not necessary for experimentation, you may want to create a new user
and authenticate the connection to your database.

For more information please check out the
[Cluster Admin Docs](http://influxdb.com/docs/v0.9/query_language/database_administration.html).

For the impatient, you can create a new admin user _bubba_ by firing off the
[InfluxDB CLI](https://github.com/influxdb/influxdb/blob/master/cmd/influx/main.go).

```shell
influx
> create user bubba with password 'bumblebeetuna'
> grant all privileges to bubba
```

And now for good measure set the credentials in you shell environment.
In the example below we will use $INFLUX_USER and $INFLUX_PWD

Now with the administrivia out of the way, let's connect to our database.

NOTE: If you've opted out of creating a user, you can omit Username and Password in
the configuration below.

```go
package main

import
import (
	"net/url"
	"fmt"
	"log"
	"os"

	"github.com/influxdb/influxdb/client/v2"
)

const (
	MyDB = "square_holes"
	username = "bubba"
	password = "bumblebeetuna"
)

func main() {
	// Make client
	u, _ := url.Parse("http://localhost:8086")
	c := client.NewClient(client.Config{
		URL: u,
		Username: username,
		Password: password,
	})

	// Create a new point batch
	bp := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  MyDB,
		Precision: "s",
	})

	// Create a point and add to batch
	tags := map[string]string{"cpu": "cpu-total"}
	fields := map[string]interface{}{
		"idle":   10.1,
		"system": 53.3,
		"user":   46.6,
	}
	pt := client.NewPoint("cpu_usage", tags, fields, time.Now())
	bp.AddPoint(pt)

	// Write the batch
	c.Write(bp)
}

```

### Inserting Data

Time series data aka *points* are written to the database using batch inserts.
The mechanism is to create one or more points and then create a batch aka
*batch points* and write these to a given database and series. A series is a
combination of a measurement (time/values) and a set of tags.

In this sample we will create a batch of a 1,000 points. Each point has a time and
a single value as well as 2 tags indicating a shape and color. We write these points
to a database called _square_holes_ using a measurement named _shapes_.

NOTE: You can specify a RetentionPolicy as part of the batch points. If not
provided InfluxDB will use the database _default_ retention policy.

```go
func writePoints(clnt client.Client) {
	sampleSize := 1000
	rand.Seed(42)

	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "systemstats",
		Precision: "us",
	})

	for i := 0; i < sampleSize; i++ {
		regions := []string{"us-west1", "us-west2", "us-west3", "us-east1"}
		tags := map[string]string{
			"cpu":    "cpu-total",
			"host":   fmt.Sprintf("host%d", rand.Intn(1000)),
			"region": regions[rand.Intn(len(regions))],
		}

		idle := rand.Float64() * 100.0
		fields := map[string]interface{}{
			"idle": idle,
			"busy": 100.0 - idle,
		}

		bp.AddPoint(client.NewPoint(
			"cpu_usage",
			tags,
			fields,
			time.Now(),
		))
	}

	err := clnt.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
}
```


### Querying Data

One nice advantage of using **InfluxDB** the ability to query your data using familiar
SQL constructs. In this example we can create a convenience function to query the database
as follows:

```go
// queryDB convenience function to query the database
func queryDB(clnt client.Client, cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: MyDB,
	}
	if response, err := clnt.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	}
	return res, nil
}
```

#### Creating a Database

```go
_, err := queryDB(clnt, fmt.Sprintf("CREATE DATABASE %s", MyDB))
if err != nil {
	log.Fatal(err)
}
```

#### Count Records

```go
q := fmt.Sprintf("SELECT count(%s) FROM %s", "value", MyMeasurement)
res, err := queryDB(clnt, q)
if err != nil {
	log.Fatal(err)
}
count := res[0].Series[0].Values[0][1]
log.Printf("Found a total of %v records\n", count)
```

#### Find the last 10 _shapes_ records

```go
q := fmt.Sprintf("SELECT * FROM %s LIMIT %d", MyMeasurement, 20)
res, err = queryDB(clnt, q)
if err != nil {
	log.Fatal(err)
}

for i, row := range res[0].Series[0].Values {
	t, err := time.Parse(time.RFC3339, row[0].(string))
	if err != nil {
		log.Fatal(err)
	}
	val := row[1].(string)
	log.Printf("[%2d] %s: %s\n", i, t.Format(time.Stamp), val)
}
```

### Using the UDP Client

The **InfluxDB** client also supports writing over UDP.

```go
func WriteUDP() {
	// Make client
	c := client.NewUDPClient("localhost:8089")

	// Create a new point batch
	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Precision: "s",
	})

	// Create a point and add to batch
	tags := map[string]string{"cpu": "cpu-total"}
	fields := map[string]interface{}{
		"idle":   10.1,
		"system": 53.3,
		"user":   46.6,
	}
	pt, err := client.NewPoint("cpu_usage", tags, fields, time.Now())
	if err != nil {
		panic(err.Error())
	}
	bp.AddPoint(pt)

	// Write the batch
	c.Write(bp)
}
```

## Go Docs

Please refer to
[http://godoc.org/github.com/influxdb/influxdb/client/v2](http://godoc.org/github.com/influxdb/influxdb/client/v2)
for documentation.

## See Also

You can also examine how the client library is used by the
[InfluxDB CLI](https://github.com/influxdb/influxdb/blob/master/cmd/influx/main.go).
