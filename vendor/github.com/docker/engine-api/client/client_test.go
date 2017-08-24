package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

func TestGetAPIPath(t *testing.T) {
	cases := []struct {
		v string
		p string
		q url.Values
		e string
	}{
		{"", "/containers/json", nil, "/containers/json"},
		{"", "/containers/json", url.Values{}, "/containers/json"},
		{"", "/containers/json", url.Values{"s": []string{"c"}}, "/containers/json?s=c"},
		{"1.22", "/containers/json", nil, "/v1.22/containers/json"},
		{"1.22", "/containers/json", url.Values{}, "/v1.22/containers/json"},
		{"1.22", "/containers/json", url.Values{"s": []string{"c"}}, "/v1.22/containers/json?s=c"},
		{"v1.22", "/containers/json", nil, "/v1.22/containers/json"},
		{"v1.22", "/containers/json", url.Values{}, "/v1.22/containers/json"},
		{"v1.22", "/containers/json", url.Values{"s": []string{"c"}}, "/v1.22/containers/json?s=c"},
		{"v1.22", "/networks/kiwl$%^", nil, "/v1.22/networks/kiwl$%25%5E"},
	}

	for _, cs := range cases {
		c, err := NewClient("unix:///var/run/docker.sock", cs.v, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		g := c.getAPIPath(cs.p, cs.q)
		if g != cs.e {
			t.Fatalf("Expected %s, got %s", cs.e, g)
		}
	}
}

func TestParseHost(t *testing.T) {
	cases := []struct {
		host  string
		proto string
		addr  string
		base  string
		err   bool
	}{
		{"", "", "", "", true},
		{"foobar", "", "", "", true},
		{"foo://bar", "foo", "bar", "", false},
		{"tcp://localhost:2476", "tcp", "localhost:2476", "", false},
		{"tcp://localhost:2476/path", "tcp", "localhost:2476", "/path", false},
	}

	for _, cs := range cases {
		p, a, b, e := ParseHost(cs.host)
		if cs.err && e == nil {
			t.Fatalf("expected error, got nil")
		}
		if !cs.err && e != nil {
			t.Fatal(e)
		}
		if cs.proto != p {
			t.Fatalf("expected proto %s, got %s", cs.proto, p)
		}
		if cs.addr != a {
			t.Fatalf("expected addr %s, got %s", cs.addr, a)
		}
		if cs.base != b {
			t.Fatalf("expected base %s, got %s", cs.base, b)
		}
	}
}

func TestUpdateClientVersion(t *testing.T) {
	client := &Client{
		transport: newMockClient(nil, func(req *http.Request) (*http.Response, error) {
			splitQuery := strings.Split(req.URL.Path, "/")
			queryVersion := splitQuery[1]
			b, err := json.Marshal(types.Version{
				APIVersion: queryVersion,
			})
			if err != nil {
				return nil, err
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(b)),
			}, nil
		}),
	}

	cases := []struct {
		v string
	}{
		{"1.20"},
		{"v1.21"},
		{"1.22"},
		{"v1.22"},
	}

	for _, cs := range cases {
		client.UpdateClientVersion(cs.v)
		r, err := client.ServerVersion(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimPrefix(r.APIVersion, "v") != strings.TrimPrefix(cs.v, "v") {
			t.Fatalf("Expected %s, got %s", cs.v, r.APIVersion)
		}
	}
}
