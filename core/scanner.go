package core

import (
	"bufio"
	"io"
)

const maxLineScannerTokenSize = 8 * 1024 * 1024

// NewLineScanner returns a scanner configured for large JSONL-style records.
func NewLineScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLineScannerTokenSize)
	return scanner
}
