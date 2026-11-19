package snowflake

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// TestErrorCodeRegistry 固化本包预置错误码的数值与输出标识。
func TestErrorCodeRegistry(t *testing.T) {
	cases := []struct {
		code       *apperr.ErrorCode
		wantNumber apperr.Code
		wantOutput apperr.Output
	}{
		{NodeInvalid, 50026, "SNOWFLAKE_NODE_INVALID"},
		{ClockBackwards, 50027, "SNOWFLAKE_CLOCK_BACKWARDS"},
	}
	for _, item := range cases {
		if item.code == nil {
			t.Fatal("错误码不得为 nil")
		}
		if item.code.GetCode() != item.wantNumber {
			t.Errorf("错误码数值不符: got %d want %d", item.code.GetCode(), item.wantNumber)
		}
		if item.code.GetOutput() != item.wantOutput {
			t.Errorf("输出标识不符: got %s want %s", item.code.GetOutput(), item.wantOutput)
		}
	}
}

// TestNewNode_InvalidParam_UsesNodeInvalidCode 固化参数越界归 50026 且保留根因。
func TestNewNode_InvalidParam_UsesNodeInvalidCode(t *testing.T) {
	cases := [][2]int64{{-1, 0}, {8, 0}, {0, -1}, {0, 8}, {99, 99}}
	for _, item := range cases {
		node, err := NewNode(item[0], item[1])
		if err == nil {
			t.Fatalf("NewNode(%d, %d) 必须返回错误", item[0], item[1])
		}
		if node != nil {
			t.Errorf("NewNode(%d, %d) 失败时不得返回节点实例", item[0], item[1])
		}
		if err.Code.Code != 50026 {
			t.Errorf("NewNode(%d, %d) 错误码 = %d; want 50026", item[0], item[1], err.Code.Code)
		}
		if err.Code.Output != "SNOWFLAKE_NODE_INVALID" {
			t.Errorf("NewNode(%d, %d) 输出标识 = %s; want SNOWFLAKE_NODE_INVALID", item[0], item[1], err.Code.Output)
		}
		if err.Err == nil {
			t.Errorf("NewNode(%d, %d) 必须保留原始越界信息为根因", item[0], item[1])
		}
	}
}

// TestNewNode_Success_ReturnsRealNil 固化成功路径返回真正的 nil 错误。
func TestNewNode_Success_ReturnsRealNil(t *testing.T) {
	node, err := NewNode(1, 1)
	if err != nil {
		t.Fatalf("合法参数不得返回错误: %v", err)
	}
	if node == nil {
		t.Fatal("合法参数必须返回节点实例")
	}
}

// TestGenerate_NilNode_UsesInvalidReceiverCode 固化 nil 节点归 50019 且哨兵可达。
func TestGenerate_NilNode_UsesInvalidReceiverCode(t *testing.T) {
	var node *Node
	_, err := node.Generate()
	if err == nil {
		t.Fatal("nil 节点发号必须返回错误")
	}
	if err.Code.Code != 50019 {
		t.Errorf("错误码 = %d; want 50019", err.Code.Code)
	}
	if err.Code.Output != "VALUE_INVALID_RECEIVER" {
		t.Errorf("输出标识 = %s; want VALUE_INVALID_RECEIVER", err.Code.Output)
	}
	if !errors.Is(err, ErrNilNode) {
		t.Error("必须保留 ErrNilNode 哨兵可达")
	}
}

// TestGenerate_ClockBackwards_UsesClockCode 固化时钟回退归 50027 且哨兵可达。
func TestGenerate_ClockBackwards_UsesClockCode(t *testing.T) {
	node, err := NewNode(1, 1)
	if err != nil {
		t.Fatalf("NewNode 失败: %v", err)
	}
	var current int64 = 1700000001000
	node.clockFunc = func() int64 {
		return atomic.LoadInt64(&current)
	}
	if _, err := node.Generate(); err != nil {
		t.Fatalf("首次发号不应失败: %v", err)
	}
	atomic.StoreInt64(&current, 1700000000000)

	_, err = node.Generate()
	if err == nil {
		t.Fatal("时钟回退必须返回错误")
	}
	if err.Code.Code != 50027 {
		t.Errorf("错误码 = %d; want 50027", err.Code.Code)
	}
	if err.Code.Output != "SNOWFLAKE_CLOCK_BACKWARDS" {
		t.Errorf("输出标识 = %s; want SNOWFLAKE_CLOCK_BACKWARDS", err.Code.Output)
	}
	if !errors.Is(err, ErrClockBackwards) {
		t.Error("必须保留 ErrClockBackwards 哨兵可达")
	}
	if err.Err == nil {
		t.Error("时钟回退必须保留根因")
	}
}

// TestGenerate_GeneOutOfRange_UsesOutOfRangeCode 固化基因越界归值错误范围码。
func TestGenerate_GeneOutOfRange_UsesOutOfRangeCode(t *testing.T) {
	node, err := NewNode(1, 1)
	if err != nil {
		t.Fatalf("NewNode 失败: %v", err)
	}
	_, err = node.Generate(Gene(64))
	if err == nil {
		t.Fatal("越界基因必须返回错误")
	}
	if err.Code.Code != 40014 {
		t.Errorf("错误码 = %d; want 40014", err.Code.Code)
	}
	if err.Err == nil {
		t.Error("基因越界必须保留根因")
	}
}

// TestGenerate_Success_ReturnsRealNil 固化发号成功路径返回真正的 nil 错误。
func TestGenerate_Success_ReturnsRealNil(t *testing.T) {
	node, err := NewNode(1, 1)
	if err != nil {
		t.Fatalf("NewNode 失败: %v", err)
	}
	if _, err := node.Generate(GeneOrder); err != nil {
		t.Fatalf("合法发号不得返回错误: %v", err)
	}
}

// TestSnowflakeID_NilReceiver_UsesInvalidReceiverCode 固化 nil 接收器归 50019。
func TestSnowflakeID_NilReceiver_UsesInvalidReceiverCode(t *testing.T) {
	var nilPtr *SnowflakeID

	err := nilPtr.UnmarshalJSON([]byte(`"1"`))
	if err == nil {
		t.Fatal("nil 接收器反序列化必须返回错误")
	}
	if !errors.Is(err, ErrNilReceiver) {
		t.Error("必须保留 ErrNilReceiver 哨兵可达")
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatal("必须可提取统一错误")
	}
	if appErr.Code.Code != 50019 {
		t.Errorf("错误码 = %d; want 50019", appErr.Code.Code)
	}

	if err := nilPtr.UnmarshalBinary([]byte("1")); !errors.Is(err, ErrNilReceiver) {
		t.Errorf("nil 接收器二进制反序列化必须保留 ErrNilReceiver: %v", err)
	}
	if err := nilPtr.Scan(int64(1)); !errors.Is(err, ErrNilReceiver) {
		t.Errorf("nil 接收器扫描必须保留 ErrNilReceiver: %v", err)
	}
}

// TestSnowflakeID_JSON_PrecisionBeyond2Pow53 固化 2^53 之外的精度防护。
func TestSnowflakeID_JSON_PrecisionBeyond2Pow53(t *testing.T) {
	// 2^53 + 1：JavaScript Number 无法精确表示该值。
	const raw int64 = 9007199254740993
	id := SnowflakeID(raw)

	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("JSON 编码失败: %v", err)
	}
	if string(data) != `"9007199254740993"` {
		t.Fatalf("JSON 输出 = %s; want \"9007199254740993\"", string(data))
	}
	if !strings.HasPrefix(string(data), `"`) {
		t.Error("JSON 必须编码为字符串以避免前端精度丢失")
	}

	var decoded SnowflakeID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON 解码失败: %v", err)
	}
	if int64(decoded) != raw {
		t.Errorf("解码结果 = %d; want %d", decoded, raw)
	}
}

// TestSnowflakeID_FormatsUnchanged 固化 JSON/Binary/SQL 的既有格式与容忍输入。
func TestSnowflakeID_FormatsUnchanged(t *testing.T) {
	id := SnowflakeID(1122334455)

	// JSON：带引号的十进制字符串。
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("JSON 编码失败: %v", err)
	}
	if string(data) != `"1122334455"` {
		t.Errorf("JSON 输出 = %s; want \"1122334455\"", string(data))
	}

	// Binary：十进制文本字节。
	binary, err := id.MarshalBinary()
	if err != nil {
		t.Fatalf("二进制编码失败: %v", err)
	}
	if string(binary) != "1122334455" {
		t.Errorf("二进制输出 = %s; want 1122334455", string(binary))
	}

	// Value：int64 原生数值。
	value, err := id.Value()
	if err != nil {
		t.Fatalf("Value 失败: %v", err)
	}
	if value != int64(1122334455) {
		t.Errorf("Value = %v; want int64(1122334455)", value)
	}

	// Scan(nil)：置零且成功。
	scanned := SnowflakeID(123)
	if err := scanned.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) 失败: %v", err)
	}
	if scanned != 0 {
		t.Errorf("Scan(nil) 后 = %d; want 0", scanned)
	}

	// Scan 容忍既有输入类型。
	for _, item := range []struct {
		input any
		want  SnowflakeID
	}{
		{int64(456), 456},
		{int(457), 457},
		{int32(458), 458},
		{[]byte("459"), 459},
		{"460", 460},
	} {
		if err := scanned.Scan(item.input); err != nil {
			t.Fatalf("Scan(%v) 失败: %v", item.input, err)
		}
		if scanned != item.want {
			t.Errorf("Scan(%v) = %d; want %d", item.input, scanned, item.want)
		}
	}
}

// TestSnowflakeID_ParseFailure_UsesInvalidFormatCode 固化解析失败归 40013。
func TestSnowflakeID_ParseFailure_UsesInvalidFormatCode(t *testing.T) {
	id := SnowflakeID(999)
	err := id.UnmarshalJSON([]byte(`"not-a-number"`))
	if err == nil {
		t.Fatal("非法字符串必须返回错误")
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatal("必须可提取统一错误")
	}
	if appErr.Code.Code != 40013 {
		t.Errorf("错误码 = %d; want 40013", appErr.Code.Code)
	}
	if appErr.Err == nil {
		t.Error("解析失败必须保留根因")
	}
	if id != 999 {
		t.Errorf("解析失败不得污染原值: got %d", id)
	}

	binary := SnowflakeID(888)
	if err := binary.UnmarshalBinary([]byte("invalid")); err == nil {
		t.Fatal("非法字节数据必须返回错误")
	}
	if binary != 888 {
		t.Errorf("解析失败不得污染原值: got %d", binary)
	}

	nullID := SnowflakeID(777)
	err = nullID.UnmarshalJSON([]byte("null"))
	if err == nil {
		t.Fatal("JSON null 必须返回错误")
	}
	var nullAppErr *apperr.Error
	if !errors.As(err, &nullAppErr) || nullAppErr.Code.Code != 40015 {
		t.Fatalf("JSON null 错误码期望 40015, 实际 %+v", nullAppErr)
	}
	if nullID != 777 {
		t.Errorf("JSON null 不得污染原值: got %d", nullID)
	}

	overflowID := SnowflakeID(666)
	err = overflowID.UnmarshalJSON([]byte(`"999999999999999999999999"`))
	if err == nil {
		t.Fatal("溢出字符串必须返回错误")
	}
	var overflowAppErr *apperr.Error
	if !errors.As(err, &overflowAppErr) || overflowAppErr.Code.Code != 40014 {
		t.Fatalf("溢出字符串错误码期望 40014, 实际 %+v", overflowAppErr)
	}
	if overflowID != 666 {
		t.Errorf("溢出解析失败不得污染原值: got %d", overflowID)
	}
}
