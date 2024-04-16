package main

import (
	"log/slog"
	"strings"

	"github.com/aofei/air"
)

func logParams(logger *slog.Logger, req *air.Request) {
	var params []string
	for _, r := range req.Params() {
		params = append(params, r.Name+"="+r.Value().String())
	}

	logger.Info("params: " + strings.Join(params, ","))
}

func getStringParam(req *air.Request, name string) string {
	p := req.Param(name)
	if p == nil {
		return ""
	}

	return p.Value().String()
}

func getBoolParam(req *air.Request, name string, def bool) bool {
	p := req.Param(name)
	if p == nil {
		return def
	}

	v, _ := p.Value().Bool()
	return v
}

func getIntParam(req *air.Request, name string, def int) int {
	p := req.Param(name)
	if p == nil {
		return def
	}

	if n, err := p.Value().Int(); err == nil {
		return n
	}

	return def
}

func getStringParamIgnoreCaps(req *air.Request, name string) string {
	nn := strings.ToLower(name)
	for _, p := range req.Params() {
		if strings.ToLower(p.Name) == nn {
			return p.Value().String()
		}
	}

	return ""
}
