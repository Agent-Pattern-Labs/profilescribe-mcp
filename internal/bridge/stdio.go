package bridge

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func readMessage(reader *bufio.Reader) ([]byte, error) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "{") {
			return []byte(trimmed), nil
		}

		contentLength, err := parseHeader(trimmed)
		if err != nil {
			return nil, err
		}

		for {
			headerLine, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			trimmedHeader := strings.TrimSpace(headerLine)
			if trimmedHeader == "" {
				break
			}

			parsedLength, err := parseHeader(trimmedHeader)
			if err != nil {
				return nil, err
			}
			if parsedLength > 0 {
				contentLength = parsedLength
			}
		}

		if contentLength <= 0 {
			return nil, fmt.Errorf("missing Content-Length header")
		}

		payload := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		return payload, nil
	}
}

func writeMessage(writer *bufio.Writer, payload []byte) error {
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
