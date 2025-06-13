package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestJWTCreateAndValidate(t *testing.T) {
	user_id := uuid.New()
	tokenSecret := "boots"

	_, err := MakeJWT(user_id, tokenSecret)
	if err != nil {
		t.Errorf("%v", err)
	}
}
