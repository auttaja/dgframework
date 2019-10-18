package main

import (
	"reflect"

	"github.com/auttaja/discordgo"
)

type CheckFunc func(interface{}) bool

type Waiter struct {
	checker    CheckFunc
	Response   chan interface{}
	eventType  reflect.Type
	removeFunc func()
}

func WaitFor(s *discordgo.Session, event interface{}, checkFunc CheckFunc) *Waiter {
	w := new(Waiter)
	w.eventType = reflect.TypeOf(event)
	w.Response = make(chan interface{}, 1)
	w.checker = checkFunc
	w.removeFunc = s.AddHandler(w.waitCallback)
	return w
}

func (w *Waiter) waitCallback(s *discordgo.Session, event interface{}) {
	if reflect.TypeOf(event).Elem() == w.eventType {
		if w.checker(event) {
			w.removeFunc()
			w.Response <- event
		}
	}
}
