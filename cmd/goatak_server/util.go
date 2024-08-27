package main

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"log/slog"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func logParams(log *slog.Logger, ctx *fiber.Ctx) {
	var params []string

	for k, v := range ctx.AllParams() {
		params = append(params, k+"="+v)
	}

	log.Info("params: " + strings.Join(params, ","))
}

// decodeMapToStruct 将 map[string]interface{} 解码为指定的结构体类型
func decodeMapToStruct[T any](m *interface{}, t *T) error {
	m2, ok := (*m).(map[string]interface{})
	if !ok {
		return fmt.Errorf("cannot convert %v to struct", reflect.TypeOf(*m))
	}
	decoderConfig := &mapstructure.DecoderConfig{
		TagName: "mapstructure",
		Result:  t,
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %v", err)
	}

	if err := decoder.Decode(m2); err != nil {
		return fmt.Errorf("failed to decode map to struct: %v", err)
	}

	return nil
}

func getStringParamIgnoreCaps(c *fiber.Ctx, name string) string {
	nn := strings.ToLower(name)
	for k, v := range c.AllParams() {
		if strings.ToLower(k) == nn {
			return v
		}
	}

	return ""
}
