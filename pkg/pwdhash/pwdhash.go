package pwdhash

import (
	"context"
	"fmt"

	"github.com/alexedwards/argon2id"
	"github.com/zeromicro/go-zero/core/logx"
	"go.uber.org/zap"
)

func HashPassword(password string) string {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		logx.WithContext(context.Background()).Error("argon2id.CreateHash error", zap.Error(err))
		panic(fmt.Sprintf("argon2id.CreateHash error: %v", err))
	}

	return hash
}

func CompareHashPassword(password, hash string) bool {
	ok, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		logx.WithContext(context.Background()).Error("argon2id.ComparePasswordAndHash error", zap.Error(err))
		panic(fmt.Sprintf("argon2id.ComparePasswordAndHash error: %v", err))
	}
	return ok
}
