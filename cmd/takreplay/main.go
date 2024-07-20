package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

func main() {
	format := flag.String("format", "", "dump format (text|json|gpx|stats)")
	uid := flag.String("uid", "", "uid to show")
	typ := flag.String("type", "", "type to show")
	n := flag.Int("n", 10, "")

	flag.Parse()

	files := flag.Args()

	if len(files) == 0 {
		fmt.Println("usage: takreplay <filename>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var dmp Dumper

	switch *format {
	case "", "text":
		dmp = new(TextDumper)
	case "json":
		dmp = new(JsonDumper)
	case "json2":
		dmp = new(Json2Dumper)
	case "gpx":
		if *uid == "" {
			fmt.Println("need uid to make gpx")
			os.Exit(1)
		}

		dmp = &GpxDumper{name: *uid}
	case "stats":
		dmp = new(StatsDumper)
	case "broadcast":
		dmp = NewBroadcastDumper(*n)
	case "contacts":
		dmp = new(ContactsDumper)
	default:
		fmt.Printf("invalid format %s\n", *format)
		os.Exit(1)
	}

	dmp.Start()

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			panic(err)
		}

		if err := readFile(f, *uid, *typ, dmp); !errors.Is(err, io.EOF) {
			fmt.Println(err)
		}
	}

	dmp.Stop()
}

func readFile(f *os.File, uid, typ string, dmp Dumper) error {
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

		if uid != "" && m.GetCotEvent().GetUid() != uid {
			continue
		}

		if typ != "" && !cot.MatchPattern(m.GetCotEvent().GetType(), typ) {
			continue
		}

		msg, err := cot.CotFromProto(m, "", "")
		if err != nil {
			return err
		}

		if err = dmp.Process(msg); err != nil {
			return err
		}
	}
}
