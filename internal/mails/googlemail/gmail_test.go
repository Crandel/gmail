package googlemail

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Crandel/gmail/internal/accounts"
)

func TestCheckOnlineError(t *testing.T) {
	t.Parallel()

	online := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer online.Close()
	if !checkOnline(online.Client(), online.URL) {
		t.Error("checkOnline() = false, want true for a 200 response")
	}

	moved := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer moved.Close()
	if !checkOnline(moved.Client(), moved.URL) {
		t.Error("checkOnline() = false, want true for a 301 response")
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	down.Close()
	if checkOnline(down.Client(), down.URL) {
		t.Error("checkOnline() = true for an unreachable server, want false")
	}
}

// TestFetchFailureReportsZero covers every failure funneling into 0; here the
// credential file is missing so no network call happens.
func TestFetchFailureReportsZero(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	acc := accounts.Account{Short: "A", MailType: accounts.Gmail, Email: "a@example.com", ClientID: "missing-client"}
	if got := FetchCount(context.Background(), acc, false); got != 0 {
		t.Errorf("FetchCount() = %d, want 0 when credentials are missing", got)
	}
}
