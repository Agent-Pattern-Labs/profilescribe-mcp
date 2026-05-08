package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunForwardsRequestsToProfileScribe(t *testing.T) {
	var sawAuth atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") == "Bearer psagt_test" {
			sawAuth.Store(true)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), `"tools/list"`) {
			t.Fatalf("request body = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`))
	}))
	defer server.Close()

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var output bytes.Buffer

	err := Run(context.Background(), Config{
		MCPURL:     server.URL,
		AgentToken: "psagt_test",
		Timeout:    time.Second,
	}, input, &output, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !sawAuth.Load() {
		t.Fatal("ProfileScribe request did not include bearer token")
	}
	if !strings.Contains(output.String(), `"result":{"tools":[]}`) {
		t.Fatalf("output = %s", output.String())
	}
}

func TestRunIgnoresNotifications(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	input := strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")
	var output bytes.Buffer

	err := Run(context.Background(), Config{
		MCPURL:     server.URL,
		AgentToken: "psagt_test",
		Timeout:    time.Second,
	}, input, &output, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("server calls = %d", calls.Load())
	}
	if output.Len() != 0 {
		t.Fatalf("output = %s", output.String())
	}
}

func TestRunWritesJSONRPCErrorForHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	input := strings.NewReader(frameContentLength(`{"jsonrpc":"2.0","id":"abc","method":"tools/list"}`))
	var output bytes.Buffer

	err := Run(context.Background(), Config{
		MCPURL:     server.URL,
		AgentToken: "psagt_test",
		Timeout:    time.Second,
	}, input, &output, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	payload := stripFrame(t, output.String())
	var response rpcResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != -32000 {
		t.Fatalf("response error = %#v", response.Error)
	}
}

func TestRunMirrorsNewlineFraming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`))
	}))
	defer server.Close()

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var output bytes.Buffer

	err := Run(context.Background(), Config{
		MCPURL:     server.URL,
		AgentToken: "psagt_test",
		Timeout:    time.Second,
	}, input, &output, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := output.String(); got != "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[]}}\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestRunWritesParseError(t *testing.T) {
	var output bytes.Buffer
	err := Run(context.Background(), Config{
		MCPURL:     "http://127.0.0.1:9",
		AgentToken: "psagt_test",
		Timeout:    time.Second,
	}, strings.NewReader("{bad json}\n"), &output, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	payload := strings.TrimSpace(output.String())
	var response rpcResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != -32700 {
		t.Fatalf("response error = %#v", response.Error)
	}
}

func frameContentLength(payload string) string {
	return "Content-Length: " + strconv.Itoa(len(payload)) + "\r\n\r\n" + payload
}

func stripFrame(t *testing.T, framed string) string {
	t.Helper()
	parts := strings.SplitN(framed, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid frame = %q", framed)
	}
	return parts[1]
}
