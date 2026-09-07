package simply

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
)

const maxDebugBody = 64 << 10

// debugTransport logs bounded request and response exchanges without logging credentials.
type debugTransport struct {
	base   http.RoundTripper
	logger *slog.Logger
}

// RoundTrip sends one HTTP exchange and records its sanitized wire details.
func (t debugTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	requestDump, requestErr := httputil.DumpRequestOut(request, true)
	if requestErr == nil {
		t.logger.Info("http request", "message", sanitizeDebugDump(requestDump))
	}
	response, err := t.base.RoundTrip(request)
	if err != nil {
		t.logger.Error("http exchange failed", "error", err)
		return nil, err
	}
	responseDump, dumpErr := httputil.DumpResponse(response, true)
	if dumpErr == nil {
		t.logger.Info("http response", "message", sanitizeDebugDump(responseDump))
	}
	return response, nil
}

// newDebugHTTPClient wraps the SDK client with a redacting structured logger.
func newDebugHTTPClient(base *http.Client, writer io.Writer) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	clone := *base
	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = debugTransport{base: transport, logger: slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	return &clone
}

// sanitizeDebugDump removes credential-bearing headers and bounds captured traffic.
func sanitizeDebugDump(dump []byte) string {
	lines := strings.Split(string(dump), "\r\n")
	for index, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "authorization:") || strings.HasPrefix(strings.ToLower(line), "cookie:") {
			name := line[:strings.IndexByte(line, ':')]
			lines[index] = name + ": [REDACTED]"
		}
	}
	result := strings.Join(lines, "\r\n")
	if len(result) <= maxDebugBody {
		return result
	}
	return result[:maxDebugBody] + fmt.Sprintf("\n...[truncated %d bytes]", len(result)-maxDebugBody)
}
