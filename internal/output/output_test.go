package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/kobberholm/go-simply-sdk"
)

func TestProductsJSONIsStableAndSorted(t *testing.T) {
	var buffer bytes.Buffer
	products := []sdk.Product{
		{Object: "z.example", Type: "domain"},
		{Object: "a.example", Type: "domain"},
	}
	if err := Products(&buffer, "json", products, sdk.Response{RateLimitRemaining: "19"}); err != nil {
		t.Fatal(err)
	}
	var result ProductsResult
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 2 || result.Data[0].Object != "a.example" || result.RateLimitRemaining != "19" {
		t.Fatalf("result = %+v", result)
	}
}

func TestRecordsTableIsSortedAndHandlesEmptyResults(t *testing.T) {
	var buffer bytes.Buffer
	if err := Records(&buffer, "table", []sdk.Record{{Name: "z", Type: "A"}, {Name: "a", Type: "A"}}, sdk.Response{}); err != nil {
		t.Fatal(err)
	}
	text := buffer.String()
	if strings.Index(text, "a") > strings.Index(text, "z") {
		t.Fatalf("records are not sorted: %q", text)
	}

	buffer.Reset()
	if err := Records(&buffer, "table", nil, sdk.Response{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "No DNS records found.") {
		t.Fatalf("empty output = %q", buffer.String())
	}
}
