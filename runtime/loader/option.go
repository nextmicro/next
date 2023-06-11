package loader

import (
	"context"
)

type Options struct {
	// for alternative data
	Context context.Context
}

type Option func(o *Options)
