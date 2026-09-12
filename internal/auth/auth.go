package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error){
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil{
		return "",err
	}
	return hash, nil
}


func CheckPasswordHash(password, hash string) (bool, error){
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil{
		return false,err
	}
	return match,nil
}


func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){
	claims := jwt.RegisteredClaims{
		Issuer: "chirpy-access",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject: userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil{
		return "",err
	}
	return tokenString, nil
}


func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error){
			if token.Method != jwt.SigningMethodHS256{
				return uuid.Nil, fmt.Errorf("invalid signing method used")
			}
			return []byte(tokenSecret), nil
		},
	)

	if err != nil{
		return uuid.Nil, err
	}

	if !token.Valid{
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil{
		return uuid.Nil, fmt.Errorf("error parsing uuid")
	}
	return userId, nil
	
}


func GetBearerToken(headers http.Header) (string, error){
	value := headers.Get("Authorization")
	if value == ""{
		return "", fmt.Errorf("no authorization header")
	}

	token_value := strings.TrimPrefix(value, "Bearer ")
	
	return token_value,nil
}


func MakeRefreshToken() string{
	bytes := make([]byte, 32)

	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}


func GetApiKey(headers http.Header) (string, error){
	value := headers.Get("Authorization")
	if value == ""{
		return "", fmt.Errorf("no authorization header")
	}

	api_value := strings.TrimPrefix(value, "ApiKey ")
	
	return api_value,nil
}