package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"goatac/cot"
	"goatac/xml"
)

type Handler struct {
	conn     net.Conn
	callsign string
	uid      string
}

func main() {
	var call = flag.String("name", "miner", "callsign")
	var addr = flag.String("addr", "127.0.0.1:8089", "host:port to connect")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	go run(ctx, *addr, *call)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	cancel()
}

func run(ctx context.Context, addr, callsign string) {
	for ctx.Err() == nil {
		fmt.Println("connecting...")
		if err := NewHandler(callsign).Start(ctx, addr); err != nil {
			time.Sleep(time.Second * 5)
		}
		fmt.Println("disconnected")
	}
}

func NewHandler(callsign string) *Handler {
	h := md5.New()
	h.Write([]byte(callsign))
	uid := fmt.Sprintf("%x", h.Sum(nil))
	uid = uid[len(uid)-14:]

	return &Handler{
		callsign: callsign,
		uid:      "ANDROID-" + uid,
	}
}

func (h *Handler) Start(ctx context.Context, addr string) error {
	var err error

	h.conn, err = net.Dial("tcp", addr)

	if err != nil {
		return err
	}

	fmt.Printf("connected with uid %s\n", h.uid)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go h.read(ctx, wg)
	go h.write(ctx, wg)
	wg.Wait()
	return nil
}

func (h *Handler) read(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	f, _ := os.Create(h.callsign + ".out")
	n := 0
	dec := xml.NewDecoder(io.TeeReader(h.conn, f))

	for ctx.Err() == nil {
		evt := &cot.Event{}
		if err := dec.Decode(evt); err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("err: %v\n", err)
			break
		}

		ProcessEvent(evt)
	}
	h.conn.Close()

	fmt.Printf("got %d messages\n", n)
}

func (h *Handler) write(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	var n int = 0

	if err := h.send(cot.MakePos(h.uid, h.callsign)); err != nil {
		panic(err)
	}

	for ctx.Err() == nil {
		var ev *cot.Event

		if n%10 == 0 {
			ev = cot.MakePos(h.uid, h.callsign)
			ev.Point.Lat = 59.9 + (rand.Float64()-0.5)/5
			ev.Point.Lon = 30.3 + (rand.Float64()-0.5)/5
			ev.Point.Hae = 20

			fmt.Println("send pos")
		} else {
			ev = cot.MakePing(h.uid)
			fmt.Println("send ping")
		}

		if ev != nil {
			if err := h.send(ev); err != nil {
				break
			}
		}
		time.Sleep(time.Second * 5)
	}
	h.conn.Close()
}

func (h *Handler) send(evt *cot.Event) error {
	if evt == nil {
		return nil
	}

	dat, err := xml.Marshal(evt)

	if err != nil {
		return err
	}

	//if _, err := h.conn.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")); err != nil {
	//	h.stop()
	//	return err
	//}
	if _, err := h.conn.Write(dat); err != nil {
		h.stop()
		return err
	}

	return nil
}

func (h *Handler) stop() {
	h.conn.Close()
}

func ProcessEvent(evt *cot.Event) {
	switch {
	case evt.Type == "t-x-c-t":
		fmt.Printf("ping from %s\n", evt.Uid)
	case evt.IsChat():
		fmt.Printf("message from %s chat %s: %s\n", evt.Detail.Chat.Sender, evt.Detail.Chat.Room, evt.GetText())
	default:
		fmt.Printf("event: %s\n", evt)
	}
}
