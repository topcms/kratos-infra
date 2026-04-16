# kratos-infra

`kratos-infra` is a shared infrastructure library example for Kratos microservices.

## Packages

- `middleware/auth`: bearer token authentication middleware
- `middleware/logging`: request logging middleware
- `middleware/tracing`: request tracing middleware (request ID)
- `middleware/recovery`: panic recovery middleware
- `errors`: shared business error definitions
- `registry`: service registry helpers
- `redis`: Redis client wrapper
- `mysql`: MySQL connection wrapper
- `kafka`: Kafka producer wrapper

## Notes

- Services should import this repo as a versioned Go module.
- Keep reusable infra logic here, not in every service repo.
