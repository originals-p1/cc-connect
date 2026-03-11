package core

import (
	"bufio"
	"errors"
	"io"
)

const (
	agentLineHardLimit = 1024 * 1024
	agentLineSoftLimit = agentLineHardLimit * 85 / 100
)

var (
	ErrAgentLineSoftLimit = errors.New("agent output line approached buffer limit")
	ErrAutoCompressNeeded = errors.New("automatic compression required")
)

// AgentLineReader reads newline-delimited agent output with a bounded line size.
// If one line reaches the soft limit, the remainder of that line is discarded and
// ErrAgentLineSoftLimit is returned so the caller can trigger recovery logic.
type AgentLineReader struct {
	reader *bufio.Reader
}

func NewAgentLineReader(r io.Reader) *AgentLineReader {
	return &AgentLineReader{
		reader: bufio.NewReaderSize(r, 64*1024),
	}
}

func (r *AgentLineReader) ReadLine() (string, error) {
	buf := make([]byte, 0, 64*1024)

	for {
		chunk, err := r.reader.ReadSlice('\n')
		if len(chunk) > 0 {
			nextLen := len(buf) + len(chunk)
			if nextLen >= agentLineSoftLimit {
				if err == bufio.ErrBufferFull {
					_ = r.discardUntilNewline()
				}
				return "", ErrAgentLineSoftLimit
			}
			buf = append(buf, chunk...)
		}

		switch err {
		case nil:
			if n := len(buf); n > 0 && buf[n-1] == '\n' {
				buf = buf[:n-1]
			}
			if n := len(buf); n > 0 && buf[n-1] == '\r' {
				buf = buf[:n-1]
			}
			return string(buf), nil
		case bufio.ErrBufferFull:
			if len(buf) >= agentLineHardLimit {
				_ = r.discardUntilNewline()
				return "", ErrAgentLineSoftLimit
			}
			continue
		case io.EOF:
			if len(buf) == 0 {
				return "", io.EOF
			}
			return string(buf), nil
		default:
			return "", err
		}
	}
}

func (r *AgentLineReader) discardUntilNewline() error {
	for {
		_, err := r.reader.ReadSlice('\n')
		if err == nil || err == io.EOF {
			return nil
		}
		if err != bufio.ErrBufferFull {
			return err
		}
	}
}
