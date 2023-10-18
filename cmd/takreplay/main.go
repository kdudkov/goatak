package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"google.golang.org/protobuf/proto"
)

func main() {
	var format = flag.String("format", "", "dump format")
	var uid = flag.String("uid", "", "uid to show")

	flag.Parse()

	file := flag.Arg(0)

	if file == "" {
		os.Exit(1)
	}

	f, err := os.Open(file)

	if err != nil {
		panic(err)
	}

	var dmp Dumper

	switch *format {
	case "":
		dmp = new(TextDumper)
	case "json":
		dmp = new(JsonDumper)
	case "gpx":
		if *uid == "" {
			fmt.Println("need uid to make gpx")
			os.Exit(1)
		}
		dmp = &GpxDumper{name: *uid}
	default:
		fmt.Printf("invalid format %s\n", format)
		os.Exit(1)
	}

	if err := readFile(f, *uid, dmp); err != io.EOF {
		fmt.Println(err)
	}
}

func readFile(f *os.File, uid string, dmp Dumper) error {
	dmp.Start()
	defer dmp.Stop()

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

		d, err := cot.DetailsFromString(m.GetCotEvent().GetDetail().GetXmlDetail())
		msg := &cot.CotMessage{TakMessage: m, Detail: d}
		if err = dmp.Process(msg); err != nil {
			return err
		}
	}
}
