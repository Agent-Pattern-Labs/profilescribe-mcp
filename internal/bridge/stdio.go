package bridge

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type messageFraming int

const (
	contentLengthFraming messageFraming = iota
	newlineFraming
)

type incomingMessage struct {
	payload []byte
	framing messageFraming
}

func readMessage(reader *bufio.Reader) (incomingMessage, error) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return incomingMessage{}, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "{") {
			return incomingMessage{payload: []byte(trimmed), framing: newlineFraming}, nil
		}

		contentLength, err := parseHeader(trimmed)
		if err != nil {
			return incomingMessage{}, err
		}

		for {
			headerLine, err := reader.ReadString('\n')
			if err != nil {
				return incomingMessage{}, err
			}
			trimmedHeader := strings.TrimSpace(headerLine)
			if trimmedHeader == "" {
				break
			}

			parsedLength, err := parseHeader(trimmedHeader)
			if err != nil {
				return incomingMessage{}, err
			}
			if parsedLength > 0 {
				contentLength = parsedLength
			}
		}

		if contentLength <= 0 {
			return incomingMessage{}, fmt.Errorf("missing Content-Length header")
		}

		payload := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return incomingMessage{}, err
		}
		return incomingMessage{payload: payload, framing: contentLengthFraming}, nil
	}
}

func writeMessage(writer *bufio.Writer, payload []byte, framing messageFraming) error {
	if framing == newlineFraming {
		payload = bytes.TrimSpace(payload)
		if _, err := writer.Write(payload); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		return writer.Flush()
	}

	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	return writer.Flush()
}

func parseHeader(line string) (int, error) {
	if !strings.Contains(line, ":") {
		return 0, fmt.Errorf("invalid header %q", line)
	}
	if !strings.EqualFold(headerName(line), "Content-Length") {
		return 0, nil
	}
	contentLength, err := strconv.Atoi(strings.TrimSpace(headerValue(line)))
	if err != nil || contentLength <= 0 {
		return 0, fmt.Errorf("invalid Content-Length %q", headerValue(line))
	}
	return contentLength, nil
}

func headerName(line string) string {
	name, _, _ := strings.Cut(line, ":")
	return strings.TrimSpace(name)
}

func headerValue(line string) string {
	_, value, _ := strings.Cut(line, ":")
	return value
}
