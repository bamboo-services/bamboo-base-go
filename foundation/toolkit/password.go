package toolkit

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// EncryptPassword 将密码加密并返回加密后的字节切片及可能发生的错误。
//
// 该函数先对明文密码进行 Base64 标准编码，再使用 bcrypt
// （代价因子为 bcrypt.DefaultCost）哈希。Base64 预编码是本包的固定
// 约定，[VerifyPassword] 与 [IsPasswordValid] 会自动保持一致。
// 明文经 Base64 编码后不得超过 72 字节（即原密码不超过 54 字节），
// 否则 bcrypt 返回密码过长错误。
//
// 加密失败时返回错误码为 [PasswordFailed]（50032）的统一错误 [*apperr.Error]，
// bcrypt 原始错误保留为根因；对外描述不含任何密码内容。
// 成功路径返回的 nil 是真正的 nil 错误。
//
// 参数说明:
//   - pass: 需要加密的明文密码。
//
// 返回值:
//   - bcrypt 哈希后的密码，可直接以字符串形式存储。
//   - 密码过长等加密过程中出现的错误。
func EncryptPassword(pass string) ([]byte, *apperr.Error) {
	// 先进行 Base64 编码
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(pass)))
	base64.StdEncoding.Encode(encoded, []byte(pass))

	// 然后使用 bcrypt 进行哈希
	hash, err := bcrypt.GenerateFromPassword(encoded, bcrypt.DefaultCost)
	if err != nil {
		return nil, apperr.New(PasswordFailed, "密码加密失败", err)
	}
	return hash, nil
}

// MustEncryptPassword 加密密码并返回加密后的字节切片。
//
// 这是 [EncryptPassword] 的便捷版本：加密失败时直接 panic，
// 仅适用于输入已保证合法的初始化阶段，一般路径请使用
// [EncryptPassword] 并处理错误。
//
// 参数说明:
//   - pass: 需要加密的明文密码。
//
// 返回值:
//   - bcrypt 哈希后的密码。
func MustEncryptPassword(pass string) []byte {
	hash, err := EncryptPassword(pass)
	if err != nil {
		panic(err)
	}
	return hash
}

// VerifyPassword 验证用户输入的密码是否与加密后的密码匹配。
//
// 验证前会先对输入明文进行 Base64 标准编码，与 [EncryptPassword]
// 的预编码约定保持一致；哈希值应传回 [EncryptPassword] 系列函数
// 产生的原始字符串。
//
// 校验失败（密码不匹配或哈希格式非法）时返回错误码为 [PasswordFailed]（50032）
// 的统一错误 [*apperr.Error]，bcrypt 原始错误保留为根因，
// 因此 errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) 仍可达；
// 对外描述不含任何密码内容。密码匹配时返回真正的 nil 错误。
//
// 参数说明:
//   - inputPass: 用户输入的明文密码。
//   - hashPass: 存储的加密密码哈希值。
//
// 返回值:
//   - 密码匹配时返回 nil；不匹配或哈希格式非法时返回携带统一错误码的错误，
//     调用方无需区分具体原因即可判定验证失败。
func VerifyPassword(inputPass, hashPass string) *apperr.Error {
	// 对输入密码进行 Base64 编码（与加密时保持一致）
	encodedInput := make([]byte, base64.StdEncoding.EncodedLen(len(inputPass)))
	base64.StdEncoding.Encode(encodedInput, []byte(inputPass))

	// 使用 bcrypt 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(hashPass), encodedInput); err != nil {
		return apperr.New(PasswordFailed, "密码校验失败", err)
	}
	return nil
}

// EncryptPasswordString 将密码加密并返回加密后的字符串。
//
// 这是 [EncryptPassword] 的字符串版本，哈希内容与错误语义完全一致，
// 便于直接落库存储：失败时返回错误码为 [PasswordFailed]（50032）的统一错误，
// 成功路径返回真正的 nil 错误。
//
// 参数说明:
//   - pass: 需要加密的明文密码。
//
// 返回值:
//   - 加密后的密码字符串。
//   - 密码过长等加密过程中出现的错误。
func EncryptPasswordString(pass string) (string, *apperr.Error) {
	hash, err := EncryptPassword(pass)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// MustEncryptPasswordString 加密密码并返回加密后的字符串。
//
// 这是 [EncryptPasswordString] 的便捷版本：加密失败时直接 panic，
// 仅适用于输入已保证合法的初始化阶段。
//
// 参数说明:
//   - pass: 需要加密的明文密码。
//
// 返回值:
//   - 加密后的密码字符串。
func MustEncryptPasswordString(pass string) string {
	hash, err := EncryptPassword(pass)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// IsPasswordValid 检查密码是否匹配，返回布尔值。
//
// 这是 [VerifyPassword] 的便捷版本：不匹配或哈希非法一律返回 false，
// 吞掉具体错误，适合仅需判定结果的调用点。
//
// 参数说明:
//   - inputPass: 用户输入的明文密码。
//   - hashPass: 存储的加密密码哈希值。
//
// 返回值:
//   - 密码匹配返回 true，否则返回 false。
func IsPasswordValid(inputPass, hashPass string) bool {
	return VerifyPassword(inputPass, hashPass) == nil
}
