package main

import "sync"

type TextLogger struct {
	lines []string
	mx    sync.RWMutex
	n     int
	cb    func()
}

func NewTextLogger() *TextLogger {
	return &TextLogger{
		lines: make([]string, 0),
		mx:    sync.RWMutex{},
		n:     100,
	}
}

func (l *TextLogger) Write(p []byte) (n int, err error) {
	if l == nil {
		return
	}
	l.AddLine(string(p))
	n = len(p)
	return
}

func (l *TextLogger) AddLine(s string) {
	if l == nil {
		return
	}
	l.mx.Lock()

	l.lines = append(l.lines, s)
	if len(l.lines) > l.n {
		l.lines = l.lines[len(l.lines)-l.n:]
	}
	l.mx.Unlock()
	if l.cb != nil {
		l.cb()
	}
}

func (l *TextLogger) AddLineColor(s string, col ...byte) {
	if l == nil {
		return
	}
	l.mx.Lock()

	l.lines = append(l.lines, WithColors(s, col...))
	if len(l.lines) > l.n {
		l.lines = l.lines[len(l.lines)-l.n:]
	}
	l.mx.Unlock()
	if l.cb != nil {
		l.cb()
	}
}

func (l *TextLogger) GetLines(n int) []string {
	if l == nil {
		return []string{}
	}
	l.mx.RLock()
	defer l.mx.RUnlock()
	if len(l.lines) <= n {
		return l.lines[:]
	}
	return l.lines[len(l.lines)-n:]
}
