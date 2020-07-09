package handlers

type MiddlewareFunc func(HandlerFunc) HandlerFunc

func (f MiddlewareFunc) Apply(next HandlerFunc) HandlerFunc {
	return f(next)
}

type Middleware interface {
	Apply(HandlerFunc) HandlerFunc
}
