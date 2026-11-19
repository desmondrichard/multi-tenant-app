package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
	OrgName        string `json:"orgName"` // <-- Added
	Role           string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, orgID, orgName, role string) (string, error) {
	hours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRY_HOURS"))
	if hours == 0 {
		hours = 72
	}
	claims := Claims{
		UserID:         userID,
		OrganizationID: orgID,
		OrgName:        orgName, // <-- Added
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(hours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func ParseToken(tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}