# Security Tooling v1.0.0

The package `github.com/Scalingo/go-utils/security` aims at providing common security helpers (e.g. token generation).

## Token Manager

```go
// Generate a time-limited token hashed with HMAC-SHA256 for the given payload.
tokenManager, _ := NewTokenManager("MY SECRET", 6 * time.Hour)
token, _ := tokenManager.GenerateToken(ctx, "payload to hash")

// Check that a hash is valid
tokenManager.CheckToken(ctx, "1663333454", "payload to hash", "a user-provided hash")
```
