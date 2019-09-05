package router

// BuildTestFunc builds a wraps the given func in the routes middleware
func (r *Route) BuildTestFunc(handler HandlerFunc) HandlerFunc {
	nhandler := handler
	for _, v := range r.Middleware {
		nhandler = v(nhandler)
	}

	return nhandler
}
