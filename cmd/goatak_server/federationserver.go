package main

import (
	"context"
	"github.com/kdudkov/goatak/internal/client"
	"log/slog"
	"net"
	"strings"
)

// fed server 作为服务器，主要工作是为客户端提供数据，但客户端不会向服务端传输数据，如有此需求应当配置双向连接

func (app *App) ListenTcpFed(ctx context.Context, addr string) (err error) {
	app.logger.Info("listening TCP Federation at " + addr)
	defer func() {
		if r := recover(); r != nil {
			app.logger.Error("panic in ListenTCP", slog.Any("error", r))
		}
	}()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		app.logger.Error("Failed to listen", slog.Any("error", err))

		return err
	}

	defer listener.Close()

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			app.logger.Error("Unable to accept connections", slog.Any("error", err))

			return err
		}

		remoteAddr := conn.RemoteAddr().String()
		app.logger.Info("TCP Federation connection from " + remoteAddr)
		h := client.NewConnClientHandler(
			conn.RemoteAddr().Network()+"_"+remoteAddr,
			conn, &client.HandlerConfig{
				// 创建一个处理客户端请求的功能，不要接收来自它的消息，但需要把消息发给它
				MessageCb:    app.DummyHandler,
				RemoveCb:     app.RemoveHandlerCb,
				NewContactCb: app.NewContactCb,
				DropMetric:   dropMetric,
				DisableRecv:  true,
				Name:         "fed_" + strings.Split(remoteAddr, ":")[0],
			})
		app.AddClientHandler(h)
		h.Start()
	}

	return nil
}
