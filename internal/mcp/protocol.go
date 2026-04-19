package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type stdioCodec struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

func newStdioCodec(r io.Reader, w io.Writer) *stdioCodec {
	return &stdioCodec{reader: bufio.NewReader(r), writer: w}
}

func (c *stdioCodec) readMessage() ([]byte, error) {
	contentLength := -1
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}

		lower := strings.ToLower(trimmed)
		if !strings.HasPrefix(lower, "content-length:") {
			continue
		}
		value := strings.TrimSpace(trimmed[len("Content-Length:"):])
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("parse content-length: %w", err)
		}
		contentLength = n
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing content-length header")
	}
	if contentLength == 0 {
		return []byte("{}"), nil
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	return payload, nil
}

func (c *stdioCodec) writeMessage(payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	headers := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := io.Copy(c.writer, bytes.NewBufferString(headers)); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	if _, err := c.writer.Write(body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return nil
}
