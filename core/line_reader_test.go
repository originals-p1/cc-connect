package core

import (
	"errors"
	"strings"
	"testing"
)

func TestLineReaderReadsNormalLine(t *testing.T) {
	reader := NewAgentLineReader(strings.NewReader("hello\n"))

	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine() error = %v", err)
	}
	if line != "hello" {
		t.Fatalf("ReadLine() = %q, want %q", line, "hello")
	}
}

func TestLineReaderStopsAtSoftLimitAndDiscardsOversizedLine(t *testing.T) {
	oversized := strings.Repeat("x", agentLineSoftLimit) + "\nnext\n"
	reader := NewAgentLineReader(strings.NewReader(oversized))

	line, err := reader.ReadLine()
	if !errors.Is(err, ErrAgentLineSoftLimit) {
		t.Fatalf("ReadLine() error = %v, want %v", err, ErrAgentLineSoftLimit)
	}
	if line != "" {
		t.Fatalf("ReadLine() line = %q, want empty", line)
	}

	line, err = reader.ReadLine()
	if err != nil {
		t.Fatalf("second ReadLine() error = %v", err)
	}
	if line != "next" {
		t.Fatalf("second ReadLine() = %q, want %q", line, "next")
	}
}
