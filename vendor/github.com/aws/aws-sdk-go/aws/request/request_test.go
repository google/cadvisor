package request_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting"
	"github.com/aws/aws-sdk-go/private/protocol/rest"
)

type testData struct {
	Data string
}

func body(str string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader([]byte(str)))
}

func unmarshal(req *request.Request) {
	defer req.HTTPResponse.Body.Close()
	if req.Data != nil {
		json.NewDecoder(req.HTTPResponse.Body).Decode(req.Data)
	}
	return
}

func unmarshalError(req *request.Request) {
	bodyBytes, err := ioutil.ReadAll(req.HTTPResponse.Body)
	if err != nil {
		req.Error = awserr.New("UnmarshaleError", req.HTTPResponse.Status, err)
		return
	}
	if len(bodyBytes) == 0 {
		req.Error = awserr.NewRequestFailure(
			awserr.New("UnmarshaleError", req.HTTPResponse.Status, fmt.Errorf("empty body")),
			req.HTTPResponse.StatusCode,
			"",
		)
		return
	}
	var jsonErr jsonErrorResponse
	if err := json.Unmarshal(bodyBytes, &jsonErr); err != nil {
		req.Error = awserr.New("UnmarshaleError", "JSON unmarshal", err)
		return
	}
	req.Error = awserr.NewRequestFailure(
		awserr.New(jsonErr.Code, jsonErr.Message, nil),
		req.HTTPResponse.StatusCode,
		"",
	)
}

type jsonErrorResponse struct {
	Code    string `json:"__type"`
	Message string `json:"message"`
}

// test that retries occur for 5xx status codes
func TestRequestRecoverRetry5xx(t *testing.T) {
	reqNum := 0
	reqs := []http.Response{
		{StatusCode: 500, Body: body(`{"__type":"UnknownError","message":"An error occurred."}`)},
		{StatusCode: 501, Body: body(`{"__type":"UnknownError","message":"An error occurred."}`)},
		{StatusCode: 200, Body: body(`{"data":"valid"}`)},
	}

	s := awstesting.NewClient(aws.NewConfig().WithMaxRetries(10))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = &reqs[reqNum]
		reqNum++
	})
	out := &testData{}
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, out)
	err := r.Send()
	assert.Nil(t, err)
	assert.Equal(t, 2, int(r.RetryCount))
	assert.Equal(t, "valid", out.Data)
}

// test that retries occur for 4xx status codes with a response type that can be retried - see `shouldRetry`
func TestRequestRecoverRetry4xxRetryable(t *testing.T) {
	reqNum := 0
	reqs := []http.Response{
		{StatusCode: 400, Body: body(`{"__type":"Throttling","message":"Rate exceeded."}`)},
		{StatusCode: 429, Body: body(`{"__type":"ProvisionedThroughputExceededException","message":"Rate exceeded."}`)},
		{StatusCode: 200, Body: body(`{"data":"valid"}`)},
	}

	s := awstesting.NewClient(aws.NewConfig().WithMaxRetries(10))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = &reqs[reqNum]
		reqNum++
	})
	out := &testData{}
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, out)
	err := r.Send()
	assert.Nil(t, err)
	assert.Equal(t, 2, int(r.RetryCount))
	assert.Equal(t, "valid", out.Data)
}

// test that retries don't occur for 4xx status codes with a response type that can't be retried
func TestRequest4xxUnretryable(t *testing.T) {
	s := awstesting.NewClient(aws.NewConfig().WithMaxRetries(10))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = &http.Response{StatusCode: 401, Body: body(`{"__type":"SignatureDoesNotMatch","message":"Signature does not match."}`)}
	})
	out := &testData{}
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, out)
	err := r.Send()
	assert.NotNil(t, err)
	if e, ok := err.(awserr.RequestFailure); ok {
		assert.Equal(t, 401, e.StatusCode())
	} else {
		assert.Fail(t, "Expected error to be a service failure")
	}
	assert.Equal(t, "SignatureDoesNotMatch", err.(awserr.Error).Code())
	assert.Equal(t, "Signature does not match.", err.(awserr.Error).Message())
	assert.Equal(t, 0, int(r.RetryCount))
}

func TestRequestExhaustRetries(t *testing.T) {
	delays := []time.Duration{}
	sleepDelay := func(delay time.Duration) {
		delays = append(delays, delay)
	}

	reqNum := 0
	reqs := []http.Response{
		{StatusCode: 500, Body: body(`{"__type":"UnknownError","message":"An error occurred."}`)},
		{StatusCode: 500, Body: body(`{"__type":"UnknownError","message":"An error occurred."}`)},
		{StatusCode: 500, Body: body(`{"__type":"UnknownError","message":"An error occurred."}`)},
		{StatusCode: 500, Body: body(`{"__type":"UnknownError","message":"An error occurred."}`)},
	}

	s := awstesting.NewClient(aws.NewConfig().WithSleepDelay(sleepDelay))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = &reqs[reqNum]
		reqNum++
	})
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, nil)
	err := r.Send()
	assert.NotNil(t, err)
	if e, ok := err.(awserr.RequestFailure); ok {
		assert.Equal(t, 500, e.StatusCode())
	} else {
		assert.Fail(t, "Expected error to be a service failure")
	}
	assert.Equal(t, "UnknownError", err.(awserr.Error).Code())
	assert.Equal(t, "An error occurred.", err.(awserr.Error).Message())
	assert.Equal(t, 3, int(r.RetryCount))

	expectDelays := []struct{ min, max time.Duration }{{30, 59}, {60, 118}, {120, 236}}
	for i, v := range delays {
		min := expectDelays[i].min * time.Millisecond
		max := expectDelays[i].max * time.Millisecond
		assert.True(t, min <= v && v <= max,
			"Expect delay to be within range, i:%d, v:%s, min:%s, max:%s", i, v, min, max)
	}
}

// test that the request is retried after the credentials are expired.
func TestRequestRecoverExpiredCreds(t *testing.T) {
	reqNum := 0
	reqs := []http.Response{
		{StatusCode: 400, Body: body(`{"__type":"ExpiredTokenException","message":"expired token"}`)},
		{StatusCode: 200, Body: body(`{"data":"valid"}`)},
	}

	s := awstesting.NewClient(&aws.Config{MaxRetries: aws.Int(10), Credentials: credentials.NewStaticCredentials("AKID", "SECRET", "")})
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)

	credExpiredBeforeRetry := false
	credExpiredAfterRetry := false

	s.Handlers.AfterRetry.PushBack(func(r *request.Request) {
		credExpiredAfterRetry = r.Config.Credentials.IsExpired()
	})

	s.Handlers.Sign.Clear()
	s.Handlers.Sign.PushBack(func(r *request.Request) {
		r.Config.Credentials.Get()
	})
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = &reqs[reqNum]
		reqNum++
	})
	out := &testData{}
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, out)
	err := r.Send()
	assert.Nil(t, err)

	assert.False(t, credExpiredBeforeRetry, "Expect valid creds before retry check")
	assert.True(t, credExpiredAfterRetry, "Expect expired creds after retry check")
	assert.False(t, s.Config.Credentials.IsExpired(), "Expect valid creds after cred expired recovery")

	assert.Equal(t, 1, int(r.RetryCount))
	assert.Equal(t, "valid", out.Data)
}

func TestMakeAddtoUserAgentHandler(t *testing.T) {
	fn := request.MakeAddToUserAgentHandler("name", "version", "extra1", "extra2")
	r := &request.Request{HTTPRequest: &http.Request{Header: http.Header{}}}
	r.HTTPRequest.Header.Set("User-Agent", "foo/bar")
	fn(r)

	assert.Equal(t, "foo/bar name/version (extra1; extra2)", r.HTTPRequest.Header.Get("User-Agent"))
}

func TestMakeAddtoUserAgentFreeFormHandler(t *testing.T) {
	fn := request.MakeAddToUserAgentFreeFormHandler("name/version (extra1; extra2)")
	r := &request.Request{HTTPRequest: &http.Request{Header: http.Header{}}}
	r.HTTPRequest.Header.Set("User-Agent", "foo/bar")
	fn(r)

	assert.Equal(t, "foo/bar name/version (extra1; extra2)", r.HTTPRequest.Header.Get("User-Agent"))
}

func TestRequestUserAgent(t *testing.T) {
	s := awstesting.NewClient(&aws.Config{Region: aws.String("us-east-1")})
	//	s.Handlers.Validate.Clear()

	req := s.NewRequest(&request.Operation{Name: "Operation"}, nil, &testData{})
	req.HTTPRequest.Header.Set("User-Agent", "foo/bar")
	assert.NoError(t, req.Build())

	expectUA := fmt.Sprintf("foo/bar %s/%s (%s; %s; %s)",
		aws.SDKName, aws.SDKVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	assert.Equal(t, expectUA, req.HTTPRequest.Header.Get("User-Agent"))
}

func TestRequestThrottleRetries(t *testing.T) {
	delays := []time.Duration{}
	sleepDelay := func(delay time.Duration) {
		delays = append(delays, delay)
	}

	reqNum := 0
	reqs := []http.Response{
		{StatusCode: 500, Body: body(`{"__type":"Throttling","message":"An error occurred."}`)},
		{StatusCode: 500, Body: body(`{"__type":"Throttling","message":"An error occurred."}`)},
		{StatusCode: 500, Body: body(`{"__type":"Throttling","message":"An error occurred."}`)},
		{StatusCode: 500, Body: body(`{"__type":"Throttling","message":"An error occurred."}`)},
	}

	s := awstesting.NewClient(aws.NewConfig().WithSleepDelay(sleepDelay))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = &reqs[reqNum]
		reqNum++
	})
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, nil)
	err := r.Send()
	assert.NotNil(t, err)
	if e, ok := err.(awserr.RequestFailure); ok {
		assert.Equal(t, 500, e.StatusCode())
	} else {
		assert.Fail(t, "Expected error to be a service failure")
	}
	assert.Equal(t, "Throttling", err.(awserr.Error).Code())
	assert.Equal(t, "An error occurred.", err.(awserr.Error).Message())
	assert.Equal(t, 3, int(r.RetryCount))

	expectDelays := []struct{ min, max time.Duration }{{500, 999}, {1000, 1998}, {2000, 3996}}
	for i, v := range delays {
		min := expectDelays[i].min * time.Millisecond
		max := expectDelays[i].max * time.Millisecond
		assert.True(t, min <= v && v <= max,
			"Expect delay to be within range, i:%d, v:%s, min:%s, max:%s", i, v, min, max)
	}
}

// test that retries occur for request timeouts when response.Body can be nil
func TestRequestRecoverTimeoutWithNilBody(t *testing.T) {
	reqNum := 0
	reqs := []*http.Response{
		{StatusCode: 0, Body: nil}, // body can be nil when requests time out
		{StatusCode: 200, Body: body(`{"data":"valid"}`)},
	}
	errors := []error{
		errTimeout, nil,
	}

	s := awstesting.NewClient(aws.NewConfig().WithMaxRetries(10))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.AfterRetry.Clear() // force retry on all errors
	s.Handlers.AfterRetry.PushBack(func(r *request.Request) {
		if r.Error != nil {
			r.Error = nil
			r.Retryable = aws.Bool(true)
			r.RetryCount++
		}
	})
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = reqs[reqNum]
		r.Error = errors[reqNum]
		reqNum++
	})
	out := &testData{}
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, out)
	err := r.Send()
	assert.Nil(t, err)
	assert.Equal(t, 1, int(r.RetryCount))
	assert.Equal(t, "valid", out.Data)
}

func TestRequestRecoverTimeoutWithNilResponse(t *testing.T) {
	reqNum := 0
	reqs := []*http.Response{
		nil,
		{StatusCode: 200, Body: body(`{"data":"valid"}`)},
	}
	errors := []error{
		errTimeout,
		nil,
	}

	s := awstesting.NewClient(aws.NewConfig().WithMaxRetries(10))
	s.Handlers.Validate.Clear()
	s.Handlers.Unmarshal.PushBack(unmarshal)
	s.Handlers.UnmarshalError.PushBack(unmarshalError)
	s.Handlers.AfterRetry.Clear() // force retry on all errors
	s.Handlers.AfterRetry.PushBack(func(r *request.Request) {
		if r.Error != nil {
			r.Error = nil
			r.Retryable = aws.Bool(true)
			r.RetryCount++
		}
	})
	s.Handlers.Send.Clear() // mock sending
	s.Handlers.Send.PushBack(func(r *request.Request) {
		r.HTTPResponse = reqs[reqNum]
		r.Error = errors[reqNum]
		reqNum++
	})
	out := &testData{}
	r := s.NewRequest(&request.Operation{Name: "Operation"}, nil, out)
	err := r.Send()
	assert.Nil(t, err)
	assert.Equal(t, 1, int(r.RetryCount))
	assert.Equal(t, "valid", out.Data)
}

func TestRequest_NoBody(t *testing.T) {
	cases := []string{
		"GET", "HEAD", "DELETE",
		"PUT", "POST", "PATCH",
	}

	for i, c := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v := r.TransferEncoding; len(v) > 0 {
				t.Errorf("%d, expect no body sent with Transfer-Encoding, %v", i, v)
			}

			outMsg := []byte(`{"Value": "abc"}`)

			if b, err := ioutil.ReadAll(r.Body); err != nil {
				t.Fatalf("%d, expect no error reading request body, got %v", i, err)
			} else if n := len(b); n > 0 {
				t.Errorf("%d, expect no request body, got %d bytes", i, n)
			}

			w.Header().Set("Content-Length", strconv.Itoa(len(outMsg)))
			if _, err := w.Write(outMsg); err != nil {
				t.Fatalf("%d, expect no error writing server response, got %v", i, err)
			}
		}))

		s := awstesting.NewClient(&aws.Config{
			Region:     aws.String("mock-region"),
			MaxRetries: aws.Int(0),
			Endpoint:   aws.String(server.URL),
			DisableSSL: aws.Bool(true),
		})
		s.Handlers.Build.PushBack(rest.Build)
		s.Handlers.Validate.Clear()
		s.Handlers.Unmarshal.PushBack(unmarshal)
		s.Handlers.UnmarshalError.PushBack(unmarshalError)

		in := struct {
			Bucket *string `location:"uri" locationName:"bucket"`
			Key    *string `location:"uri" locationName:"key"`
		}{
			Bucket: aws.String("mybucket"), Key: aws.String("myKey"),
		}

		out := struct {
			Value *string
		}{}

		r := s.NewRequest(&request.Operation{
			Name: "OpName", HTTPMethod: c, HTTPPath: "/{bucket}/{key+}",
		}, &in, &out)

		if err := r.Send(); err != nil {
			t.Fatalf("%d, expect no error sending request, got %v", i, err)
		}
	}
}
