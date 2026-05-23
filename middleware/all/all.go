// Package all 通过 blank import 一次性注册 next 框架内置的全部 middleware factory
// 到 middleware.DefaultRegistry。
//
// 业务方在 main.go 中只需一行：
//
//	import _ "github.com/nextmicro/next/middleware/all"
//
// 即可启用 tracing / recovery / metrics / metadata / logging / bbr / circuitbreaker。
//
// 拆分到独立包的原因：
//   - 部分 factory 依赖平台相关库（如 bbr 通过 aegis 访问 cgroup），
//     在某些环境（如 macOS 单元测试）会在 init 阶段触发 panic。
//   - 业务可按需精细化选择：例如只导入 tracing/recovery 子包，跳过 bbr。
package all

import (
	_ "github.com/nextmicro/next/middleware/bbr"
	_ "github.com/nextmicro/next/middleware/circuitbreaker"
	_ "github.com/nextmicro/next/middleware/logging"
	_ "github.com/nextmicro/next/middleware/metadata"
	_ "github.com/nextmicro/next/middleware/metrics"
	_ "github.com/nextmicro/next/middleware/recovery"
	_ "github.com/nextmicro/next/middleware/tracing"
)
