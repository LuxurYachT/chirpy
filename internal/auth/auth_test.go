package auth

import (
	"testing"
	"github.com/google/uuid"
	"time"
)

func TestJWTCreateAndValidate(t *testing.T) {
	user_id := uuid.New()
	tokenSecret := "boots"
	expires, err := time.ParseDuration("24h")

	_, err = MakeJWT(user_id, tokenSecret, expires)
	if err != nil {
		t.Errorf("%v", err)
	}
}