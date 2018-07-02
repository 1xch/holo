package engine

type HandleErrorFunc func(*Engine, error)

func defaultHandleError(e *Engine, r error) {
	if r != nil {
		e.last = r
		e.kill = true
	}
}

type ErrorHandler struct {
	e    *Engine
	hefn HandleErrorFunc
	last error
	warn []error
}

func (e *ErrorHandler) Init(n *Engine) {
	e.e = n
	e.last = nil
	e.hefn = defaultHandleError
}

func (e *ErrorHandler) SetHandleError(fn HandleErrorFunc) {
	e.hefn = fn
}

func (e *ErrorHandler) HandleError(r error) {
	e.hefn(e.e, r)
}

func (e *ErrorHandler) HandleWarning(w ...error) {
	for _, r := range w {
		e.warn = append(e.warn, r)
		e.e.Println(r)
	}
}
