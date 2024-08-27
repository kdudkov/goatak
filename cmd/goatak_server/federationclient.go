package main

import (
	"context"
	"fmt"
	"github.com/kdudkov/goatak/internal/client"
	"log/slog"
	"sync"
	"time"
)

// ConnectToFedServer 这是Fed服务器，只应该从服务器获取信息，但不要发送任何信息过去
func (app *App) ConnectToFedServer(ctx context.Context, fed *FedConfig) {
	for ctx.Err() == nil {
		addr := fmt.Sprintf("%s:%d:%s", fed.Host, fed.Port, fed.Proto) // localhost:8087:tcp
		conn, err := app.connect(addr)
		if err != nil {
			app.logger.Error("Fed Server connect error", slog.Any("error", err))
			time.Sleep(time.Second * 5)
			continue
		}

		fedName := fmt.Sprintf("fed_%s:%v", fed.Host, fed.Port)
		app.logger.Info(fmt.Sprintf("Federation to %s connected", fedName))

		wg := &sync.WaitGroup{}
		wg.Add(1)

		h := client.NewConnClientHandler(addr, conn, &client.HandlerConfig{
			MessageCb: app.NewCotMessage,
			RemoveCb: func(ch client.ClientHandler) {
				wg.Done()
				app.handlers.Delete(addr)
				app.logger.Info("disconnected")
			},
			NewContactCb: app.NewContactCb,
			Name:         fedName,
			DisableSend:  true,
			IsClient:     true,
			UID:          app.uid,
		})

		go h.Start()
		app.AddClientHandler(h)

		wg.Wait()
	}
}

func (app *App) connToFed(ctx context.Context, fed *FedConfig) error {
	app.ConnectToFedServer(ctx, fed)
	return nil
}
