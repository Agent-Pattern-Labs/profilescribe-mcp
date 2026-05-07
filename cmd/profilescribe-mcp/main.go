package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/razroo/profilescribe-mcp/internal/bridge"
)

func main() {
	logger := log.New(os.Stderr, "profilescribe-mcp: ", 0)

	if err := bridge.Run(context.Background(), bridge.ConfigFromEnv(), os.Stdin, os.Stdout, logger); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return
		}
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
