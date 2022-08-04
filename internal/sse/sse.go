package sse

import (
	"fmt"
	"io"
	"net/http"
)

type Event struct {
	Event string
	Data  []byte
}

// writeHeader write Server-Send Event related headers to the rw.
func WriteHeader(rw http.ResponseWriter) {
	rw.Header().Add("Content-Type", "text/event-stream")
	rw.Header().Add("Cache-Control", "no-store")
	rw.Header().Add("X-Accel-Buffering", "no")
	rw.WriteHeader(http.StatusOK)
}

// Send write a Server-Send Event with event and data field to w.
// If w is http.Flusher, writeEvent will flush the writer, ensure the client
// receive immediately.
func (ev *Event) Send(w io.Writer) error {
	eventLine := fmt.Sprintf("event:%s\n", ev.Event)
	_, err := w.Write([]byte(eventLine))
	if err != nil {
		return err
	}

	if len(ev.Data) > 0 {
		dataLine := fmt.Sprintf("data:%s\n\n", string(ev.Data))
		_, err = w.Write([]byte(dataLine))
		if err != nil {
			return err
		}
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}
