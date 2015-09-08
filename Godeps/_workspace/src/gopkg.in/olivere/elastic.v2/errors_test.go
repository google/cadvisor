package elastic

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestResponseError(t *testing.T) {
	message := "Something went seriously wrong."
	raw := "HTTP/1.1 500 Internal Server Error\r\n" +
		"\r\n" +
		`{"status":500,"error":"` + message + `"}` + "\r\n"
	r := bufio.NewReader(strings.NewReader(raw))

	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = checkResponse(resp)
	if err == nil {
		t.Fatalf("expected error; got: %v", err)
	}

	// Check for correct error message
	expected := fmt.Sprintf("elastic: Error %d (%s): %s", resp.StatusCode, http.StatusText(resp.StatusCode), message)
	got := err.Error()
	if got != expected {
		t.Fatalf("expected %q; got: %q", expected, got)
	}

	// Check that error is of type *elastic.Error, which contains additional information
	e, ok := err.(*Error)
	if !ok {
		t.Fatal("expected error to be of type *elastic.Error")
	}
	if e.Status != resp.StatusCode {
		t.Fatalf("expected status code %d; got: %d", resp.StatusCode, e.Status)
	}
	if e.Message != message {
		t.Fatalf("expected error message %q; got: %q", message, e.Message)
	}
}

func TestResponseErrorHTML(t *testing.T) {
	raw := "HTTP/1.1 413 Request Entity Too Large\r\n" +
		"\r\n" +
		`<html>
<head><title>413 Request Entity Too Large</title></head>
<body bgcolor="white">
<center><h1>413 Request Entity Too Large</h1></center>
<hr><center>nginx/1.6.2</center>
</body>
</html>` + "\r\n"
	r := bufio.NewReader(strings.NewReader(raw))

	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = checkResponse(resp)
	if err == nil {
		t.Fatalf("expected error; got: %v", err)
	}

	// Check for correct error message
	expected := fmt.Sprintf("elastic: Error %d (%s)", http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge))
	got := err.Error()
	if got != expected {
		t.Fatalf("expected %q; got: %q", expected, got)
	}
}
