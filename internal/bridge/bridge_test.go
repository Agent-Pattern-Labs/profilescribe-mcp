package bridge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestRunAdvertisesImagePathForUploadProfileImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"upload_profile_image","description":"Upload base64-encoded profile or header image bytes.","inputSchema":{"type":"object","required":["kind","imageBase64"],"properties":{"kind":{"type":"string","enum":["profile","header"]},"imageBase64":{"type":"string"}}}}]}}`))
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

	var response struct {
		Result struct {
			Tools []struct {
				InputSchema struct {
					Required   []string                  `json:"required"`
					Properties map[string]map[string]any `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Result.Tools) != 1 {
		t.Fatalf("tools = %#v", response.Result.Tools)
	}
	if got := strings.Join(response.Result.Tools[0].InputSchema.Required, ","); got != "kind" {
		t.Fatalf("required = %q", got)
	}
	if _, ok := response.Result.Tools[0].InputSchema.Properties["imagePath"]; !ok {
		t.Fatalf("imagePath was not advertised: %#v", response.Result.Tools[0].InputSchema.Properties)
	}
}

func TestRunExpandsUploadProfileImagePath(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "avatar.png")
	if err := os.WriteFile(imagePath, testPNG(), 0o600); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	var sawImageBase64 atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			Params struct {
				Name      string `json:"name"`
				Arguments struct {
					Kind        string `json:"kind"`
					ImageBase64 string `json:"imageBase64"`
					ImagePath   string `json:"imagePath"`
				} `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Method != "tools/call" || request.Params.Name != "upload_profile_image" {
			t.Fatalf("unexpected request = %#v", request)
		}
		if request.Params.Arguments.ImagePath != "" {
			t.Fatalf("imagePath should not be forwarded")
		}
		content, err := base64.StdEncoding.DecodeString(request.Params.Arguments.ImageBase64)
		if err != nil {
			t.Fatalf("decode imageBase64: %v", err)
		}
		if !bytes.Equal(content, testPNG()) {
			t.Fatalf("imageBase64 content mismatch")
		}
		sawImageBase64.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{}"}]}}`))
	}))
	defer server.Close()

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"upload_profile_image","arguments":{"kind":"profile","imagePath":"` + imagePath + `"}}}` + "\n")
	var output bytes.Buffer

	err := Run(context.Background(), Config{
		MCPURL:     server.URL,
		AgentToken: "psagt_test",
		Timeout:    time.Second,
	}, input, &output, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !sawImageBase64.Load() {
		t.Fatal("ProfileScribe request did not include imageBase64")
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

func testPNG() []byte {
	content, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	return content
}
