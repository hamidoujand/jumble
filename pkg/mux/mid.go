package mux

// middleware represents a middleware in our custom mux.
type Middleware func(HandlerFunc) HandlerFunc

func registerMiddlewares(h HandlerFunc, mids []Middleware) HandlerFunc {
	//apply them from last to first around the handler

	for i := len(mids) - 1; i >= 0; i-- {
		m := mids[i]
		if m != nil {
			h = m(h)
		}
	}

	return h
}
