package gpsd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"
)

const (
	DefaultAddress = "localhost:2947"
	DialTimeout    = time.Millisecond * 500
)

type BaseMsg struct {
	Class string `json:"class"`
}

type TPVMsg struct {
	Class  string    `json:"class"`
	Tag    string    `json:"tag"`
	Device string    `json:"device"`
	Mode   int       `json:"mode"`
	Time   time.Time `json:"time"`
	Ept    float64   `json:"ept"`
	Lat    float64   `json:"lat"`
	Lon    float64   `json:"lon"`
	Alt    float64   `json:"alt"`
	Epx    float64   `json:"epx"`
	Epy    float64   `json:"epy"`
	Epv    float64   `json:"epv"`
	Track  float64   `json:"track"`
	Speed  float64   `json:"speed"`
	Climb  float64   `json:"climb"`
	Epd    float64   `json:"epd"`
	Eps    float64   `json:"eps"`
	Epc    float64   `json:"epc"`
	Eph    float64   `json:"eph"`
}

type VERSIONMsg struct {
	Class      string `json:"class"`
	Release    string `json:"release"`
	Rev        string `json:"rev"`
	ProtoMajor int    `json:"proto_major"`
	ProtoMinor int    `json:"proto_minor"`
	Remote     string `json:"remote"`
}

type GpsdClient struct {
	addr   string
	conn   net.Conn
	logger *slog.Logger
	reader *bufio.Reader
}

func New(addr string, logger *slog.Logger) *GpsdClient {
	c := &GpsdClient{
		addr:   DefaultAddress,
		conn:   nil,
		logger: logger,
		reader: nil,
	}

	if addr != "" {
		c.addr = addr
	}

	return c
}

func (c *GpsdClient) connect(ctx context.Context) bool {
	timeout := time.Second * 5

	for {
		conn, err := net.DialTimeout("tcp4", c.addr, DialTimeout)

		if err == nil {
			c.conn = conn
			c.reader = bufio.NewReader(c.conn)

			_, _ = fmt.Fprintf(c.conn, "?WATCH={\"enable\":true,\"json\":true}")

			return true
		}

		c.logger.Error("dial error", "error", err)

		select {
		case <-time.After(timeout):
		case <-ctx.Done():
			c.logger.Error("stop connection attempts")
			return false
		}

		if timeout < time.Minute {
			timeout = timeout * 2
		}
	}
}

func (c *GpsdClient) Listen(ctx context.Context, cb func(lat, lon, alt, speed, track float64)) {
	for ctx.Err() == nil {
		if c.conn == nil {
			if !c.connect(ctx) {
				return
			}
		}

		line, err := c.reader.ReadString('\n')

		if err != nil {
			c.logger.Error("error", "error", err)

			_ = c.conn.Close()
			c.conn = nil
			continue
		}

		data := []byte(line)

		var msg BaseMsg

		if err1 := json.Unmarshal(data, &msg); err1 != nil {
			c.logger.Error("JSON decode error", "error", err1)
			c.logger.Debug("bad json: " + line)
			_ = c.conn.Close()
			c.conn = nil
			continue
		}

		switch msg.Class {
		case "TPV":
			var r *TPVMsg
			if err1 := json.Unmarshal(data, &r); err1 != nil {
				c.logger.Error("JSON decode error", "error", err1)
			}

			if cb != nil {
				cb(r.Lat, r.Lon, r.Alt, r.Speed, r.Track)
			}
		case "VERSION":
			var r *VERSIONMsg
			if err1 := json.Unmarshal(data, &r); err1 != nil {
				c.logger.Error("JSON decode error", "error", err1)
			}
			c.logger.Info(fmt.Sprintf("got version %s, rev. %s", r.Release, r.Rev))
		}
	}
}
