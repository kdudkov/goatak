package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"gotac/cot"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	conn net.Conn
	buf  *bytes.Buffer
}

func main() {
	for {
		fmt.Println("connecting...")
		if err := NewHandler().Start("127.0.0.1:8089"); err != nil {
			time.Sleep(time.Second * 5)
		}
		//conn, err := net.Dial("tcp", "204.48.30.216:8087")
		fmt.Println("disconnected")
	}

	//c := make(chan os.Signal, 1)
	//signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	//<-c

	//ioutil.WriteFile("a.out", b.Bytes(), 0x666)
}

func NewHandler() *Handler {
	return &Handler{
		buf:  new(bytes.Buffer),
	}
}

func (h *Handler) Start(addr string) error {
	var err error

	h.conn, err = net.Dial("tcp", addr)
	//conn, err := net.Dial("tcp", "204.48.30.216:8087")

	if err != nil {
		return err
	}

	fmt.Println("connected")

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go h.read(wg)
	go h.write(wg)
	wg.Wait()
	return nil
}

func (h *Handler) read(wg *sync.WaitGroup) {
	defer wg.Done()

	dec := xml.NewDecoder(io.TeeReader(h.conn, h.buf))
	for {
		// Read tokens from the XML document in a stream.
		t, _ := dec.Token()
		if t == nil {
			h.stop()
			return
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "event" {
				var ev cot.Event
				dec.DecodeElement(&ev, &se)
				ProcessEvent(&ev)
			}
		case xml.CharData:
			continue
		case xml.ProcInst:
			continue
		default:
			fmt.Printf("%s\n", reflect.TypeOf(t).Name())
		}
	}
}

func (h *Handler) write(wg *sync.WaitGroup) {
	defer wg.Done()

	var n int = 0
	var uid = "ANDROID-aabbcc5578"

	for {
		if n%10 == 0 {
			ev := cot.MakePos(uid, "miner")
			if _, err := h.conn.Write([]byte(ev)); err != nil {
				h.stop()
				return
			}
			fmt.Println("pos")
		} else {
			ev := cot.MakePing(uid)
			if _, err := h.conn.Write([]byte(ev)); err != nil {
				h.stop()
				return
			}
			fmt.Println("ping")
		}

		time.Sleep(time.Second * 5)
		n++
	}
}

func (h *Handler) stop () {
	h.conn.Close()
}

func ProcessEvent(evt *cot.Event) {
	switch {
	case evt.Type == "t-x-c-t":
		// ping
	case evt.Type == "a-f-G-U-C":
		// position
		fmt.Printf("status from %s (%s) %s %s\n", evt.Uid, evt.GetCallsign(), evt.Detail.TakVersion.Device, evt.Detail.TakVersion.Platform)
	case evt.Type == "a-f-G-U-C-I":
		// position
		fmt.Printf("I status from %s (%s) %s %s\n", evt.Uid, evt.GetCallsign(), evt.Detail.TakVersion.Device, evt.Detail.TakVersion.Platform)
	case strings.HasPrefix(evt.Type, "b-m-p-w-"):
		fmt.Printf("add point %s (%s)\n", evt.Uid, evt.Detail.Contact.Callsign)
	case evt.IsChat():
		fmt.Printf("message from %s chat %s: %s\n", evt.Detail.Chat.Sender, evt.Detail.Chat.Room, evt.GetText())
	default:
		fmt.Printf("event: %s/%s (%s)\n", evt.Uid, evt.Type, evt.GetCallsign())
	}
}
