package healthz

import (
	"net/http"
)

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Register simple HTTP /healthz handler to return "ok".
func RegisterHandler() error {
	http.HandleFunc("/healthz", handleHealthz)
	return nil
}
