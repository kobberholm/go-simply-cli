package simply

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientBearerAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "key", AuthMode: "bearer", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Products().List(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewClientBasicAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("account:key"))
		if r.Header.Get("Authorization") != expected {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "key", Account: "account", AuthMode: "basic", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Products().List(context.Background()); err != nil {
		t.Fatal(err)
	}
}
