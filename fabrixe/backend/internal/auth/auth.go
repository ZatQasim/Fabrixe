package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Role constants
const (
	RoleAdministrator = "administrator"
	RoleOperator      = "operator"
	RoleViewer        = "viewer"
)

// Claims defines the JWT payload.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"sub"`
	Role     string `json:"role"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// TokenPair holds an access + refresh token pair.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Service handles JWT operations.
type Service struct {
	secret         []byte
	accessTokenTTL time.Duration
}

// New creates an Auth service.
func New(secret string, accessTokenTTLMinutes int) *Service {
	return &Service{
		secret:         []byte(secret),
		accessTokenTTL: time.Duration(accessTokenTTLMinutes) * time.Minute,
	}
}

// IssueTokenPair creates a new access + refresh token pair for a user.
func (s *Service) IssueTokenPair(userID int64, username, role string) (*TokenPair, error) {
	sessionID := uuid.NewString()
	expiresAt := time.Now().Add(s.accessTokenTTL)

	claims := &Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "fabrixe",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshToken := uuid.NewString() + "-" + uuid.NewString()

	return &TokenPair{
		AccessToken:  signed,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken parses and validates a JWT access token.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// RoleWeight returns a numeric weight for role comparison.
func RoleWeight(role string) int {
	switch role {
	case RoleAdministrator:
		return 3
	case RoleOperator:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// HasPermission checks that a role meets the minimum required role level.
func HasPermission(userRole, requiredRole string) bool {
	return RoleWeight(userRole) >= RoleWeight(requiredRole)
}

var (
	ErrTokenExpired = errors.New("token has expired")
	ErrTokenInvalid = errors.New("token is invalid")
)
