package dgframework

import (
	"reflect"

	"github.com/auttaja/discordgo"
)

// CheckFunc represents a generic check function to wait for
type CheckFunc func(interface{}) bool

// Waiter represents a single event you'd like to wait for
type Waiter struct {
	checker    CheckFunc
	Response   chan interface{}
	eventType  reflect.Type
	removeFunc func()
}

// WaitFor submits a request to wait for an event.  You are responsible for listening to the channel to get the data
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
