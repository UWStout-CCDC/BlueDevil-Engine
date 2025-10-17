package webpages

// Context key exported for storing/retrieving the authenticated user from request.Context
type ContextKey string

var CtxUserKey ContextKey = "user"
