package relay

// Provider identifies a communications provider independent of channel.
type Provider interface {
	Name() string
}
