package context

import (
	"net/http"
)

// Context is a custom context that carries a reference to the Client.
type Context struct {
	Client *http.Client
	Req    *http.Request
	Next   func(*Context) (*http.Response, error)
}

// NewContext creates a new Context.
func NewContext(client *http.Client, req *http.Request) *Context {
	return &Context{
		Client: client,
		Req:    req,
		Next:   nil,
	}
}
