package router

import (
	"regexp"
	"strings"

	"github.com/auttaja/discordgo"
)

// HandlerFunc is a command handler
type HandlerFunc func(*Context) error

// MiddlewareFunc is a middleware
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// NewRegexMatcher returns a new regex matcher
func NewRegexMatcher(regex string) func(string) bool {
	r := regexp.MustCompile(regex)
	return func(command string) bool {
		return r.MatchString(command)
	}
}

// NewNameMatcher returns a matcher that matches a route's name and aliases
func NewNameMatcher(r *Route) func(string) bool {
	return func(command string) bool {
		for _, v := range r.Aliases {
			if command == v {
				return true
			}
		}
		return command == r.Name
	}
}

// Group allows you to do things like more easily manage categories
// For example, setting the routes category in the callback will cause
// All future added routes to inherit the category.
// example:
// Group(func (r *Route) {
//    r.Cat("stuff")
//    r.On("thing", nil).Desc("the category of this function will be stuff")
// })
func (r *Route) Group(fn func(r *Route)) *Route {
	rt := New()
	fn(rt)
	for _, v := range rt.Routes {
		r.AddRoute(v)
	}
	return r
}

// Use adds the given middleware func to this route's middleware chain
func (r *Route) Use(fn ...MiddlewareFunc) *Route {
	r.Middleware = append(r.Middleware, fn...)
	return r
}

// On registers a route with the name you supply
//    name    : name of the route to create
//    handler : handler function
func (r *Route) On(name string, handler HandlerFunc) *Route {
	rt := r.OnMatch(name, nil, handler)
	rt.Matcher = NewNameMatcher(rt)
	return rt
}

// OnMatch adds a handler for the given route
//    name    : name of the route to add
//    matcher : matcher function used to match the route
//    handler : handler function for the route
func (r *Route) OnMatch(name string, matcher func(string) bool, handler HandlerFunc) *Route {
	if rt := r.Find(name); rt != nil {
		return rt
	}

	nhandler := handler

	// Add middleware to the handler
	for _, v := range r.Middleware {
		nhandler = v(nhandler)
	}

	rt := &Route{
		Name:     name,
		Category: r.Category,
		Handler:  nhandler,
		Matcher:  matcher,
	}

	r.AddRoute(rt)
	return rt
}

// AddRoute adds a route to the router
// Will return RouteAlreadyExists error on failure
//    route : route to add
func (r *Route) AddRoute(route *Route) error {
	// Check if the route already exists
	if rt := r.Find(route.Name); rt != nil {
		return ErrRouteAlreadyExists
	}

	route.Parent = r
	r.Routes = append(r.Routes, route)
	return nil
}

// RemoveRoute removes a route from the router
//     route : route to remove
func (r *Route) RemoveRoute(route *Route) error {
	for i, v := range r.Routes {
		if v == route {
			r.Routes = append(r.Routes[:i], r.Routes[i+1:]...)
			return nil
		}
	}
	return ErrCouldNotFindRoute
}

// Find finds a route with the given name
// It will return nil if nothing is found
//    name : name of route to find
func (r *Route) Find(name string) *Route {
	for _, v := range r.Routes {
		if v.Matcher(name) {
			return v
		}
	}
	return nil
}

// FindFull a full path of routes by searching through their subroutes
// Until the deepest match is found.
// It will return the route matched and the depth it was found at
//     args : path of route you wish to find
//            ex. FindFull(command, subroute1, subroute2, nonexistent)
//            will return the deepest found match, which will be subroute2
func (r *Route) FindFull(args ...string) (*Route, int) {
	nr := r
	i := 0
	for _, v := range args {
		if rt := nr.Find(v); rt != nil {
			nr = rt
			i++
		} else {
			break
		}
	}
	return nr, i
}

// New returns a new route
func New() *Route {
	return &Route{
		Routes: []*Route{},
	}
}

// Route is a command route
type Route struct {
	// Routes is a slice of subroutes
	Routes []*Route

	Name        string
	Aliases     []string
	Description string
	UsageString string
	Category    string

	// Matcher is a function that determines
	// If this route will be matched
	Matcher func(string) bool

	// Handler is the Handler for this route
	Handler HandlerFunc

	// Default route for responding to bot mentions
	Default *Route

	// The parent for this route
	Parent *Route

	// Middleware to be applied when adding subroutes
	Middleware []MiddlewareFunc
}

// Desc sets this routes description
func (r *Route) Desc(description string) *Route {
	r.Description = description
	return r
}

// Usage sets the routes usage string describing how to use the command
func (r *Route) Usage(usage string) *Route {
	r.UsageString = usage
	return r
}

// Cat sets this route's category
func (r *Route) Cat(category string) *Route {
	r.Category = category
	return r
}

// Alias appends aliases to this route's alias list
func (r *Route) Alias(aliases ...string) *Route {
	r.Aliases = append(r.Aliases, aliases...)
	return r
}

func mention(id string) string {
	return "<@" + id + ">"
}

func nickMention(id string) string {
	return "<@!" + id + ">"
}

// FindAndExecute is a helper method for calling routes
// it creates a context from a message, finds its route, and executes the handler
// it looks for a message prefix which is either the prefix specified or the message is prefixed
// with a bot mention
//    s            : discordgo session to pass to context
//    prefix       : prefix you want the bot to respond to
//    botID        : user ID of the bot to allow you to substitute the bot ID for a prefix
//    m            : discord message to pass to context
func (r *Route) FindAndExecute(s *discordgo.Session, prefix string, botID string, m *discordgo.Message) error {
	var pf string

	// If the message content is only a bot mention and the mention route is not nil, send the mention route
	if r.Default != nil && m.Content == mention(botID) || r.Default != nil && m.Content == nickMention(botID) {
		_ = r.Default.Handler(NewContext(s, m, []string{""}, r.Default))
		return nil
	}

	// Append a space to the mentions
	bmention := mention(botID) + " "
	nmention := nickMention(botID) + " "

	p := func(t string) bool {
		return strings.HasPrefix(m.Content, t)
	}

	switch {
	case prefix != "" && p(prefix):
		pf = prefix
	case p(bmention):
		pf = bmention
	case p(nmention):
		pf = nmention
	default:
		return ErrCouldNotFindRoute
	}

	command := strings.TrimPrefix(m.Content, pf)
	args := ParseArgs(command)

	if rt, depth := r.FindFull(args...); depth > 0 {
		args = append([]string{strings.Join(args[:depth], string(separator))}, args[depth:]...)
		ctx := NewContext(s, m, args, rt)
		defer HandlePanic(ctx)
		err := rt.Handler(ctx)
		if err != nil {
			HandleError(ctx, err)
		}
	} else {
		return ErrCouldNotFindRoute
	}

	return nil
}
