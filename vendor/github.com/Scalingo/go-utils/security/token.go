package security

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/Scalingo/go-utils/crypto"
)

var (
	ErrInvalidTimestamp = errors.New("timestamp wrongly formatted")
	ErrFutureTimestamp  = errors.New("invalid timestamp in the future")
	ErrTokenExpired     = errors.New("token expired")
)

// TokenGenerator lets you generate a Token.
type TokenGenerator interface {
	GenerateToken(context.Context, string) (Token, error)
}

// TokenChecker checks if a given payload matches a user-provided hash.
type TokenChecker interface {
	CheckToken(ctx context.Context, timestamp, payload, hashHex string) (bool, error)
}

// Token contains a hashed payload generated at a specific time.
type Token struct {
	// GeneratedAt is the token generation date represented as a Unix time
	GeneratedAt int64
	// Hash is the hex encoded HMAC of the token
	Hash string
}

type TokenManager struct {
	tokenSecretKey []byte
	tokenValidity  time.Duration
	now            func() time.Time
}

// The TokenManager must implement the TokenGenerator and the TokenChecker interfaces.
var _ TokenGenerator = TokenManager{}
var _ TokenChecker = TokenManager{}

// NewTokenManager instantiates a new TokenGenerator with the given token configuration:
// - tokenSecretKey: secret to generate the token.
// - tokenValidity: validity duration of the token.
func NewTokenManager(tokenSecretKey []byte, tokenValidity time.Duration) TokenManager {
	return TokenManager{
		tokenSecretKey: tokenSecretKey,
		tokenValidity:  tokenValidity,
		now:            time.Now,
	}
}

// GenerateToken generates a new time-limited token hashed with HMAC-SHA256 for the given payload.
func (g TokenManager) GenerateToken(ctx context.Context, payload string) (Token, error) {
	generatedAtTimestamp := g.now().Unix()
	// Generate a hash for those metadata
	hash := crypto.HMAC256(g.tokenSecretKey, []byte(generatePlainText(generatedAtTimestamp, payload)))

	return Token{
		GeneratedAt: generatedAtTimestamp,
		Hash:        hex.EncodeToString(hash),
	}, nil
}

// CheckToken checks whether some metadata matches a user-provided hash. The metadata contains:
//   - timestamp: Unix time of the generation of the hashHex
//   - payload: payload to generate a new hash to check hashHex against
//
// hashHex is the string representation of the hex encoded HMAC-SHA256 that should be tested.
func (g TokenManager) CheckToken(ctx context.Context, timestamp, payload, hashHex string) (bool, error) {
	generatedAtTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, ErrInvalidTimestamp
	}

	generatedAt := time.Unix(generatedAtTimestamp, -1)
	// If the generatedAt timestamp is in the future => reject
	if generatedAt.After(g.now()) {
		return false, ErrFutureTimestamp
	}

	// If the generatedAt timestamp is older than the tokenValidity => reject
	if g.now().After(generatedAt.Add(g.tokenValidity)) {
		return false, ErrTokenExpired
	}

	// Try to decode the hash as an hex string
	hash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false, errors.Wrap(err, "fail to decode the hash as a valid hex representation")
	}

	// Generate a hash for the given metadata
	generatedHash := crypto.HMAC256(g.tokenSecretKey, []byte(generatePlainText(generatedAtTimestamp, payload)))

	// Compare the generated hash with the one provided by the client
	return hmac.Equal(generatedHash, hash), nil
}

func generatePlainText(timestamp int64, payload string) string {
	return fmt.Sprintf("%v/%v", timestamp, payload)
}
