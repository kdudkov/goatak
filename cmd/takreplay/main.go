package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"google.golang.org/protobuf/proto"
)

func main() {
	var file = flag.String("file", "", "record all events to file")
	var useJson = flag.Bool("json", false, "dump in json")
	flag.Parse()

	f, err := os.Open(*file)

	if err != nil {
		panic(err)
	}

	if err := readFile(f, *useJson); err != io.EOF {
		fmt.Println(err)
	}
}

func readFile(f *os.File, useJson bool) error {
	lenBuf := make([]byte, 2)
	for {
		if _, err := io.ReadFull(f, lenBuf); err != nil {
			return err
		}

		n := uint32(lenBuf[0]) + uint32(lenBuf[1])*256
		buf := make([]byte, n)

		if _, err := io.ReadFull(f, buf); err != nil {
			return err
		}

		m := new(cotproto.TakMessage)

		if err := proto.Unmarshal(buf, m); err != nil {
			return err
		}

		d, err := cot.DetailsFromString(m.GetCotEvent().GetDetail().GetXmlDetail())

		if err != nil {
			return err
		}

		if useJson {
			b, err := json.Marshal(m)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		} else {
			msg := &cot.CotMessage{TakMessage: m, Detail: d}
			fmt.Println(msg.GetSendTime().Format(time.DateTime), msg.GetType(), msg.GetCallsign())
		}
	}
}
