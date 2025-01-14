package filter

type Stream interface {
	// OnStreamComplete runs when a Stream ends, which can happen at any point in the protocol lifecycle (e.g due to an
	// ImmediateResponse being returned).
	OnStreamComplete(req *RequestContext)
}
