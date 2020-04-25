# zipkin-go-opentracing

[![Travis CI](https://travis-ci.org/openzipkin-contrib/zipkin-go-opentracing.svg?branch=master)](https://travis-ci.org/openzipkin-contrib/zipkin-go-opentracing)
[![GoDoc](https://godoc.org/github.com/openzipkin-contrib/zipkin-go-opentracing?status.svg)](https://godoc.org/github.com/openzipkin-contrib/zipkin-go-opentracing)
[![Go Report Card](https://goreportcard.com/badge/github.com/openzipkin-contrib/zipkin-go-opentracing)](https://goreportcard.com/report/github.com/openzipkin-contrib/zipkin-go-opentracing)
[![Sourcegraph](https://sourcegraph.com/github.com/openzipkin-contrib/zipkin-go-opentracing/-/badge.svg)](https://sourcegraph.com/github.com/openzipkin-contrib/zipkin-go-opentracing?badge)

[OpenTracing](http://opentracing.io) bridge for the native [Zipkin](https://zipkin.io) tracing implementation [Zipkin Go](https://github.com/openzipkin/zipkin-go).

### Notes

This package is a simple bridge to allow OpenTracing API consumers
to use Zipkin as their tracing backend. For details on how to work with spans
and traces we suggest looking at the documentation and README from the
[OpenTracing API](https://github.com/opentracing/opentracing-go).

For developers interested in adding Zipkin tracing to their Go services we
suggest looking at [Go kit](https://gokit.io) which is an excellent toolkit to
instrument your distributed system with Zipkin and much more with clean
separation of domains like transport, middleware / instrumentation and
business logic.

### Examples

Please check the [zipkin-go](https://github.com/openzipkin/zipkin-go) package for information how to set-up the Zipkin Go native tracer. Once set-up you can simple call the `Wrap` function to create the OpenTracing compatible bridge.

```go
import (
	"github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	zipkinot "github.com/openzipkin-contrib/zipkin-go-opentracing"
)

func main() {
	// bootstrap your app...
  
	// zipkin / opentracing specific stuff
	{
		// set up a span reporter
		reporter := zipkinhttp.NewReporter("http://zipkinhost:9411/api/v2/spans")
		defer reporter.Close()
  
		// create our local service endpoint
		endpoint, err := zipkin.NewEndpoint("myService", "myservice.mydomain.com:80")
		if err != nil {
			log.Fatalf("unable to create local endpoint: %+v\n", err)
		}

		// initialize our tracer
		nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
		if err != nil {
			log.Fatalf("unable to create tracer: %+v\n", err)
		}

		// use zipkin-go-opentracing to wrap our tracer
		tracer := zipkinot.Wrap(nativeTracer)
  
		// optionally set as Global OpenTracing tracer instance
		opentracing.SetGlobalTracer(tracer)
	}
  
	// do other bootstrapping stuff...
}
```

For more information on zipkin-go-opentracing, please see the documentation at
[go doc](https://godoc.org/github.com/openzipkin-contrib/zipkin-go-opentracing).
