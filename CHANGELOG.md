# Changelog

## v0.1.0 - 2026-04-16
### Added
- 新增 `middleware/tracing`，支持 `X-Request-Id` 透传
- 新增 `middleware/logging`，支持 trace_id + latency 日志
- 新增 `mysql.NewDB` / `redis.NewClient` 统一初始化

### Changed
- `registry/consul` 默认 `WaitEvery` 调整为 2s

### Fixed
- recovery panic 包装错误格式统一

### Breaking Changes
- 无

### Migration Notes
- `kratos-template` 需在中间件链按 `tracing -> logging -> recovery` 顺序接入
