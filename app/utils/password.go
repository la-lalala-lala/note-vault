package utils

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const maxBcryptPasswordBytes = 72

// HashPassword 使用 bcrypt 对明文密码进行不可逆哈希。
// bcrypt 最多处理 72 字节密码，调用方不能把返回值以外的明文写入数据库或日志。
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("密码不能为空")
	}
	if len([]byte(password)) > maxBcryptPasswordBytes {
		return "", fmt.Errorf("密码不能超过 %d 字节", maxBcryptPasswordBytes)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword 使用 bcrypt 校验明文密码是否与数据库中的哈希匹配。
// 密码不匹配返回 false 和 nil；哈希格式损坏等异常返回具体错误。
func VerifyPassword(passwordHash, password string) (bool, error) {
	if passwordHash == "" || password == "" {
		return false, nil
	}
	if len([]byte(password)) > maxBcryptPasswordBytes {
		return false, nil
	}

	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, fmt.Errorf("校验密码哈希失败: %w", err)
}
