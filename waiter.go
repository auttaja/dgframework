package main

import (
	"reflect"

	"github.com/auttaja/discordgo"
)

type CheckFunc func(interface{}) bool

type Waiter struct {
	Checker    CheckFunc
	Response   chan interface{}
	EventType  reflect.Type
	removeFunc func()
}

func WaitFor(s *discordgo.Session, event interface{}, checkFunc CheckFunc) *Waiter {
	w := new(Waiter)
	w.EventType = reflect.TypeOf(event)
	w.Response = make(chan interface{}, 1)
	w.Checker = checkFunc
	w.removeFunc = s.AddHandler(w.waitCallback)
	return w
}

func (w *Waiter) waitCallback(s *discordgo.Session, event interface{}) {
	if reflect.TypeOf(event).Elem() == w.EventType {
		if w.Checker(event) {
			w.removeFunc()
			w.Response <- event
		}
	}
}
