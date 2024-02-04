package model

type Answer[T any] struct {
	Version string `json:"version"`
	Type    string `json:"type"`
	NodeID  string `json:"nodeId"`
	Data    T      `json:"data"`
}
