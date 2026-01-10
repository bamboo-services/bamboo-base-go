package test

import (
	xUtil "github.com/bamboo-services/bamboo-base-go/utility"
	"strings"
	"testing"
)

// Test_GenerateSecurityKet 测试 GenerateLongSecurityKey 函数的正确性。
func Test_GenerateSecurityKet(t *testing.T) {
	key := xUtil.GenerateLongSecurityKey()
	t.Logf("生成的安全密钥: %s", key)
	if len(key) < 10 {
		t.Errorf("生成的安全密钥长度不足: %d", len(key))
	}
	if strings.HasPrefix(key, "cs_") == false {
		t.Errorf("生成的安全密钥前缀不正确: %s", key[:3])
	}
	if len(key) != 67 {
		t.Errorf("生成的安全密钥长度不正确: %d", len(key))
	}
}
