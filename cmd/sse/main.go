package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/yepengliang/sse-demo/internal/sse"
)

// timeStream writes current time to the client every second.
// It stops when the client close the connection.
func timeStream(rw http.ResponseWriter, req *http.Request) {
	sse.WriteHeader(rw)

	for {
		data, _ := json.Marshal(time.Now())
		event := &sse.Event{
			Event: "time",
			Data:  data,
		}

		if err := event.Send(rw); err != nil {
			log.Printf("connection err: %v\n", err)
			break
		}

		time.Sleep(1 * time.Second)
	}
}

type ClientCounter struct {
	count   int64
	clients sync.Map // map[uuid.UUID]chan int64
}

func (cc *ClientCounter) AddClient(clientID uuid.UUID) <-chan int64 {
	atomic.AddInt64(&cc.count, 1)

	ch := make(chan int64, 1)
	cc.clients.Store(clientID, ch)

	cc.Boardcast()

	return ch
}

func (cc *ClientCounter) RemoveClient(clientID uuid.UUID) {
	ch, ok := cc.clients.LoadAndDelete(clientID)
	if !ok {
		return
	}
	if ch, ok := ch.(chan int64); ok {
		close(ch)
	}

	atomic.AddInt64(&cc.count, -1)
	cc.Boardcast()
}

func (c *ClientCounter) Boardcast() {
	c.clients.Range(func(clientID, ch any) bool {
		if ch, ok := ch.(chan int64); ok {
			ch <- atomic.LoadInt64(&c.count)
		}
		return true
	})
}

func counterStream(counter *ClientCounter) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		clientID := uuid.Must(uuid.NewV4())
		pingTicker := time.NewTicker(10 * time.Second)

		ch := counter.AddClient(clientID)
		defer counter.RemoveClient(clientID)

		sse.WriteHeader(rw)

		for {
			select {
			case <-ctx.Done():
				counter.RemoveClient(clientID)
				return

			case count, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(struct{ Count int64 }{count})
				event := &sse.Event{
					Event: "client-count",
					Data:  data,
				}
				if err := event.Send(rw); err != nil {
					log.Printf("send event err: %v\n", err)
					return
				}

			case <-pingTicker.C:
				pingEvent := &sse.Event{
					Event: "ping",
				}
				if err := pingEvent.Send(rw); err != nil {
					log.Printf("ping err: %v\n", err)
					return
				}
			}
		}
	}
}

func main() {
	mux := http.NewServeMux()

	counter := &ClientCounter{}

	mux.HandleFunc("/api/time", timeStream)
	mux.HandleFunc("/api/client-count", counterStream(counter))

	err := http.ListenAndServe("127.0.0.1:3000", mux)
	if err != nil {
		log.Fatalf("http server err: %v", err)
	}
}
