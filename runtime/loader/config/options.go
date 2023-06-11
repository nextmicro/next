package config

import (
	"context"

	"github.com/nextmicro/next/runtime/loader"
)

type filePathKey struct{}

// WithPath sets the path to file
func WithPath(p string) loader.Option {
	return func(o *loader.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, filePathKey{}, p)
	}
}
