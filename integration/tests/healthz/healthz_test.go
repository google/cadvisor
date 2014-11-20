package healthz

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/cadvisor/integration/framework"
)

func TestHealthzOk(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Ensure that /heathz returns "ok"
	resp, err := http.Get(fm.Host().FullHost() + "healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if string(body) != "ok" {
		t.Fatalf("cAdvisor returned unexpected healthz status of %q", body)
	}
}
