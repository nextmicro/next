package broker

// Wrapper wraps a Broker and returns a Broker
type Wrapper func(Broker) Broker

// Chain returns a Broker that specifies the chained handler for endpoint.
func Chain(m ...Wrapper) Wrapper {
	return func(broker Broker) Broker {
		for i := len(m) - 1; i >= 0; i-- {
			broker = m[i](broker)
		}
		return broker
	}
}
