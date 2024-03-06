package loader

import (
	"context"
)

type Options struct {
	Initialized bool
	// for alternative data
	Context context.Context
}

type Option func(o *Options)

func NewOptions(opts ...Option) *Options {
	options := Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	return &options
}
