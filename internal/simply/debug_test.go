package simply

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDebugTransportRedactsAuthorization(t *testing.T) {
	var logs bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Debug", "yes")
		_, _ = writer.Write([]byte(`[]`))
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "secret-key", BaseURL: server.URL, DebugWriter: &logs})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Products().List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "secret-key") || !strings.Contains(logs.String(), "[REDACTED]") {
		t.Fatalf("logs = %q", logs.String())
	}
	if !strings.Contains(logs.String(), "http request") || !strings.Contains(logs.String(), "http response") {
		t.Fatalf("logs = %q", logs.String())
	}
}
