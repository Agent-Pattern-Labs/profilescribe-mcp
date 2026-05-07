package bridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

func Run(ctx context.Context, cfg Config, input io.Reader, output io.Writer, logger *log.Logger) error {
	cfg = cfg.normalized()
	if err := cfg.validate(); err != nil {
		return err
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	client := &http.Client{Timeout: cfg.Timeout}
	reader := bufio.NewReader(input)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	for {
		payload, err := readMessage(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(payload)) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			if err := writeMessage(writer, errorResponse(nil, -32700, "parse error")); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			logger.Printf("ignored notification %s", req.Method)
			continue
		}

		responsePayload, err := callProfileScribe(ctx, client, cfg, payload)
		if err != nil {
			if err := writeMessage(writer, errorResponse(req.ID, -32000, err.Error())); err != nil {
				return err
			}
			continue
		}
		if err := writeMessage(writer, responsePayload); err != nil {
			return err
		}
	}
}
