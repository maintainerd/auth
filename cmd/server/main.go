package main

import (
	"context"
	"log/slog"
	"os"
)

// main is intentionally tiny: all startup work lives in run so the executable
// boundary only handles process concerns like logging the final error and exit.
func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("Server startup failed", "error", err)
		os.Exit(1)
	}
}
