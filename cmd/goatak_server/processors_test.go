package main

import (
	"go.uber.org/zap"
	"testing"
)

func TestGetProcessor(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	app := NewApp(&AppConfig{}, logger.Sugar())
	app.InitMessageProcessors()

	data := map[string]string{
		"a-b-c-d":   "a-",
		"b-t-f":     "b-t-f",
		"b-t-f-a":   "b-t-f-",
		"b-t-f-a-b": "b-t-f-",
		"b-t-b-a":   "b-",
	}

	for k, v := range data {
		p, _ := app.GetProcessor(k)
		if p != v {
			t.Errorf("got %s, must be %s", p, v)
		}
	}
}
