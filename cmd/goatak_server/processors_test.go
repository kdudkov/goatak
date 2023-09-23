package main

import (
	"go.uber.org/zap"
	"testing"
)

func TestGetProcessor(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	app := NewApp(&AppConfig{}, logger.Sugar())
	app.InitMessageProcessors()

	k, _ := app.GetProcessor("a-b-c-d")
	if k != "a-" {
		t.Fail()
	}
}
