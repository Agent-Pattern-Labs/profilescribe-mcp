package bridge

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestReadMessageAcceptsNewlineJSON(t *testing.T) {
	got, err := readMessage(bufio.NewReader(strings.NewReader(`{"jsonrpc":"2.0"}` + "\n")))
	if err != nil {
		t.Fatalf("readMessage returned error: %v", err)
	}
	if string(got.payload) != `{"jsonrpc":"2.0"}` {
		t.Fatalf("payload = %q", got.payload)
	}
	if got.framing != newlineFraming {
		t.Fatalf("framing = %v", got.framing)
	}
}

func TestReadMessageAcceptsContentLengthFraming(t *testing.T) {
	payload := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	input := "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\nContent-Length: " + strconv.Itoa(len(payload)) + "\r\n\r\n" + payload

	got, err := readMessage(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("readMessage returned error: %v", err)
	}
	if string(got.payload) != payload {
		t.Fatalf("payload = %q", got.payload)
	}
	if got.framing != contentLengthFraming {
		t.Fatalf("framing = %v", got.framing)
	}
}

func TestWriteMessageUsesContentLengthFraming(t *testing.T) {
	var out bytes.Buffer
	if err := writeMessage(bufio.NewWriter(&out), []byte(`{"ok":true}`), contentLengthFraming); err != nil {
		t.Fatalf("writeMessage returned error: %v", err)
	}
	if got := out.String(); got != "Content-Length: 11\r\n\r\n{\"ok\":true}" {
		t.Fatalf("framed output = %q", got)
	}
}

func TestWriteMessageUsesNewlineFraming(t *testing.T) {
	var out bytes.Buffer
	if err := writeMessage(bufio.NewWriter(&out), []byte("{\"ok\":true}\n"), newlineFraming); err != nil {
		t.Fatalf("writeMessage returned error: %v", err)
	}
	if got := out.String(); got != "{\"ok\":true}\n" {
		t.Fatalf("framed output = %q", got)
	}
}

func TestParseHeaderRejectsInvalidContentLength(t *testing.T) {
	if _, err := parseHeader("Content-Length: nope"); err == nil {
		t.Fatal("expected invalid Content-Length error")
	}
}
