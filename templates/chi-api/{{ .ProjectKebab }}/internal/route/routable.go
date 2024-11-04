package route

type Routable interface {
	Routes() []*Route
}
