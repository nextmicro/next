package config

import (
	_ "github.com/nextmicro/next/middleware/bbr"
	_ "github.com/nextmicro/next/middleware/circuitbreaker"
	_ "github.com/nextmicro/next/middleware/logging"
	_ "github.com/nextmicro/next/middleware/metadata"
	_ "github.com/nextmicro/next/middleware/metrics"
	_ "github.com/nextmicro/next/middleware/recovery"
	_ "github.com/nextmicro/next/middleware/tracing"
)
