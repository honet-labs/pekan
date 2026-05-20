package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	TenantID    string   `json:"tid"`
	TenantCode  string   `json:"tcd"`
	SessionID   string   `json:"sid"`
	TokenType   string   `json:"typ"`
	Permissions []string `json:"permissions"`
	Features    []string `json:"features"`
	Modules     []string `json:"modules"`
	jwt.RegisteredClaims
}

type Service struct {
	issuer string
	secret []byte
	ttl    time.Duration
}

func NewService(issuer, secret string, ttl time.Duration) *Service {
	return &Service{
		issuer: issuer,
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (s *Service) ParseAccessToken(rawToken string) (Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithLeeway(30*time.Second),
	)

	token, err := parser.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return Claims{}, errors.New("invalid token")
	}
	if claims.UserID == "" || claims.TenantID == "" || claims.SessionID == "" {
		return Claims{}, errors.New("invalid token claims")
	}
	if claims.Subject != claims.UserID {
		return Claims{}, errors.New("invalid token subject")
	}

	return *claims, nil
}

type IssueAccessTokenInput struct {
	UserID      string
	Email       string
	TenantID    string
	TenantCode  string
	SessionID   string
	Permissions []string
	Features    []string
	Modules     []string
}

func (s *Service) IssueAccessToken(in IssueAccessTokenInput) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	claims := Claims{
		UserID:      in.UserID,
		Email:       in.Email,
		TenantID:    in.TenantID,
		TenantCode:  in.TenantCode,
		SessionID:   in.SessionID,
		TokenType:   "access",
		Permissions: in.Permissions,
		Features:    in.Features,
		Modules:     in.Modules,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    s.issuer,
			Subject:   in.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}
