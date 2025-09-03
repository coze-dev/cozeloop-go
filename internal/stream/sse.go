// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/coze-dev/cozeloop-go/internal/util"
)

// ServerSentEvent represents a Server-Sent Event
type ServerSentEvent struct {
	Event string
	Data  string
	ID    string
	Retry *int
}

// JSON unmarshals the Data field into the provided interface
func (sse *ServerSentEvent) JSON(v interface{}) error {
	if sse.Data == "" {
		return fmt.Errorf("empty data field")
	}
	return json.Unmarshal([]byte(sse.Data), v)
}

// SSEDecoder decodes Server-Sent Events from an io.Reader
type SSEDecoder struct {
	scanner *bufio.Scanner
}

// NewSSEDecoder creates a new SSE decoder
func NewSSEDecoder(reader io.Reader) *SSEDecoder {
	return &SSEDecoder{
		scanner: bufio.NewScanner(reader),
	}
}

// Decode decodes SSE events from the reader and returns a channel
func (d *SSEDecoder) Decode(ctx context.Context) <-chan SSEEvent {
	ch := make(chan SSEEvent, 1)

	util.GoSafe(ctx, func() {
		defer close(ch)

		for {
			event, err := d.DecodeEvent()
			if err != nil {
				if err != io.EOF {
					ch <- SSEEvent{Error: err}
				}
				return
			}

			if event != nil {
				ch <- SSEEvent{Event: event}
			}
		}
	})

	return ch
}

// SSEEvent wraps either an event or an error
type SSEEvent struct {
	Event *ServerSentEvent
	Error error
}

// DecodeEvent decodes a single SSE event
func (d *SSEDecoder) DecodeEvent() (*ServerSentEvent, error) {
	event := &ServerSentEvent{}
	var dataLines []string

	for d.scanner.Scan() {
		line := d.scanner.Text()

		// Empty line indicates end of event
		if strings.TrimSpace(line) == "" {
			if len(dataLines) > 0 || event.Event != "" || event.ID != "" || event.Retry != nil {
				event.Data = strings.Join(dataLines, "\n")
				return event, nil
			}
			continue
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			// Line without colon, treat as field name with empty value
			field := strings.TrimSpace(line)
			d.processField(event, field, "", &dataLines)
			continue
		}

		field := line[:colonIndex]
		value := line[colonIndex+1:]

		// Remove leading space from value
		if strings.HasPrefix(value, " ") {
			value = value[1:]
		}

		d.processField(event, field, value, &dataLines)
	}

	if err := d.scanner.Err(); err != nil {
		return nil, err
	}

	// If we reach here, it's EOF
	if len(dataLines) > 0 || event.Event != "" || event.ID != "" || event.Retry != nil {
		event.Data = strings.Join(dataLines, "\n")
		return event, nil
	}

	return nil, io.EOF
}

// processField processes a single SSE field
func (d *SSEDecoder) processField(event *ServerSentEvent, field, value string, dataLines *[]string) {
	switch field {
	case "event":
		event.Event = value
	case "data":
		*dataLines = append(*dataLines, value)
	case "id":
		event.ID = value
	case "retry":
		if retry, err := strconv.Atoi(value); err == nil {
			event.Retry = &retry
		}
	}
}
