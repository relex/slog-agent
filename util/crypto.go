package util

import (
	"crypto/md5" //nolint:gosec
	"crypto/sha512"
	"encoding/hex"

	"github.com/relex/gotils/logger"
)

// MD5ToHexdigest computes MD5 for given string and returns hex
func MD5ToHexdigest(content string) string {
	hasher := md5.New() //nolint:gosec
	if _, err := hasher.Write([]byte(content)); err != nil {
		logger.Panic(err)
	}
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

// SHA512ToHexdigest computes SHA512 for given string and returns hex
func SHA512ToHexdigest(content string) string {
	hasher := sha512.New()
	if _, err := hasher.Write([]byte(content)); err != nil {
		logger.Panic(err)
	}
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}
