package core

import (
	"strings"
	"testing"
)

func TestNewLineScannerHandlesLargeLines(t *testing.T) {
	line := strings.Repeat("x", 70*1024) + "\n"
	scanner := NewLineScanner(strings.NewReader(line))

	if !scanner.Scan() {
		t.Fatalf("expected scanner to read large line, err=%v", scanner.Err())
	}
	if got := scanner.Text(); len(got) != 70*1024 {
		t.Fatalf("line length = %d, want %d", len(got), 70*1024)
	}
}
