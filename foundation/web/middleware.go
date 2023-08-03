package web

type Middleware func(Handler) Handler

func withMiddleware(handler Handler, mvs ...Middleware) Handler {
	for i := len(mvs) - 1; i >= 0; i-- {
		if mv := mvs[i]; mv != nil {
			handler = mv(handler)
		}

	}
	return handler
}
