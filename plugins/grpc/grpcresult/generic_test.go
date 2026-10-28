package grpcresult_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult/internaltest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// plainStruct 用于反例测试的普通非指针非 Protobuf 结构体。
type plainStruct struct {
	Value string
}

// nonProtoStruct 用于反例测试的普通指针结构体（未实现 proto.Message）。
type nonProtoStruct struct {
	Header string
}

// assertPanic 运行 fn 并断言其发生符合预期的编程错误 panic。
// 必须同时满足：
// 1. 明确发生 panic，而非静默完成；
// 2. panic 信息必须以 "grpcresult.SuccessWith:" 开头，证明属于 SuccessWith 框架自身校验触发的编程错误；
// 3. panic 信息包含 expectedSubstring 关键提示，不将其他未知 panic 误判为测试通过。
func assertPanic(t *testing.T, expectedSubstring string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("期望发生包含 %q 的 panic，但实际未发生 panic", expectedSubstring)
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.HasPrefix(msg, "grpcresult.SuccessWith:") {
			t.Fatalf("非预期的 panic 格式（必须以 'grpcresult.SuccessWith:' 开头）: %s", msg)
		}
		if !strings.Contains(msg, expectedSubstring) {
			t.Fatalf("panic 信息未包含预期的关键描述 %q，实际为: %s", expectedSubstring, msg)
		}
	}()
	fn()
}

// TestSuccessWith_ValidAlphaResponse 测试标准 Protobuf 响应结构体 ValidAlphaResponse（字段名为 Header）。
func TestSuccessWith_ValidAlphaResponse(t *testing.T) {
	t.Run("标准非调试模式", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "trace-alpha-std-001")
		msg := "Alpha operation succeeded"

		resp := grpcresult.SuccessWith[*internaltest.ValidAlphaResponse](ctx, msg)
		if resp == nil {
			t.Fatal("期望返回非 nil 的 ValidAlphaResponse 实例")
		}
		if resp.Header == nil {
			t.Fatal("期望 Header 字段被正确注入且非 nil")
		}
		if resp.Header.Code != 200 {
			t.Errorf("期望 Code 为 200，实际为 %d", resp.Header.Code)
		}
		if resp.Header.Output != grpcresult.DefaultSuccessOutput {
			t.Errorf("期望 Output 为 %s，实际为 %s", grpcresult.DefaultSuccessOutput, resp.Header.Output)
		}
		if resp.Header.Message != msg {
			t.Errorf("期望 Message 为 %q，实际为 %q", msg, resp.Header.Message)
		}
		if resp.Header.Context != "trace-alpha-std-001" {
			t.Errorf("期望 Context 为 trace-alpha-std-001，实际为 %s", resp.Header.Context)
		}
		if resp.Header.Overhead != nil {
			t.Errorf("非调试模式下期望 Overhead 为 nil，实际为 %v", resp.Header.Overhead)
		}
		if resp.Payload != "" {
			t.Errorf("未赋值的业务载荷 Payload 期望为空字符串，实际为 %s", resp.Payload)
		}

		// 真实 Protobuf 序列化与反序列化一致性验证
		rawBytes, err := proto.Marshal(resp)
		if err != nil {
			t.Fatalf("proto.Marshal 序列化失败: %v", err)
		}
		unmarshaled := &internaltest.ValidAlphaResponse{}
		if err := proto.Unmarshal(rawBytes, unmarshaled); err != nil {
			t.Fatalf("proto.Unmarshal 反序列化失败: %v", err)
		}
		if !proto.Equal(resp, unmarshaled) {
			t.Errorf("反序列化后对象与原对象不一致: 原始=%v, 反序列化=%v", resp, unmarshaled)
		}
		if unmarshaled.Header == nil || unmarshaled.Header.Code != 200 || unmarshaled.Header.Context != "trace-alpha-std-001" {
			t.Errorf("反序列化后 Header 关键字段校验失败: %v", unmarshaled.Header)
		}
	})

	t.Run("调试模式与耗时注入", func(t *testing.T) {
		_ = os.Setenv(string(env.Debug), "true")
		defer func() { _ = os.Unsetenv(string(env.Debug)) }()

		startTime := time.Now().Add(-20 * time.Millisecond)
		ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "trace-alpha-debug-002")
		ctx = context.WithValue(ctx, ctxkey.UserStartTimeKey, startTime)

		msg := "Alpha with debug overhead"
		resp := grpcresult.SuccessWith[*internaltest.ValidAlphaResponse](ctx, msg)
		if resp == nil || resp.Header == nil {
			t.Fatal("期望返回非 nil 的响应与 Header")
		}
		if resp.Header.Context != "trace-alpha-debug-002" {
			t.Errorf("期望 Context 为 trace-alpha-debug-002，实际为 %s", resp.Header.Context)
		}
		if resp.Header.Overhead == nil || *resp.Header.Overhead <= 0 {
			t.Errorf("调试模式下期望 Overhead 注入正整数耗时微秒，实际为 %v", resp.Header.Overhead)
		}

		// 调试模式下序列化与反序列化耗时一致性
		rawBytes, err := proto.Marshal(resp)
		if err != nil {
			t.Fatalf("proto.Marshal 序列化失败: %v", err)
		}
		unmarshaled := &internaltest.ValidAlphaResponse{}
		if err := proto.Unmarshal(rawBytes, unmarshaled); err != nil {
			t.Fatalf("proto.Unmarshal 反序列化失败: %v", err)
		}
		if unmarshaled.Header == nil || unmarshaled.Header.Overhead == nil {
			t.Fatalf("反序列化后 Header 或 Overhead 丢失: %v", unmarshaled)
		}
		if *unmarshaled.Header.Overhead != *resp.Header.Overhead {
			t.Errorf("反序列化后 Overhead 不一致: 期望 %d，实际 %d", *resp.Header.Overhead, *unmarshaled.Header.Overhead)
		}
	})
}

// TestSuccessWith_ValidBetaResponse 测试自定义字段名称的 Protobuf 响应结构体 ValidBetaResponse（字段名为 CustomBase）。
func TestSuccessWith_ValidBetaResponse(t *testing.T) {
	t.Run("自定义BaseResponse字段名注入", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "trace-beta-std-001")
		msg := "Beta operation succeeded"

		resp := grpcresult.SuccessWith[*internaltest.ValidBetaResponse](ctx, msg)
		if resp == nil {
			t.Fatal("期望返回非 nil 的 ValidBetaResponse 实例")
		}
		if resp.CustomBase == nil {
			t.Fatal("期望 CustomBase 字段被正确识别注入且非 nil")
		}
		if resp.CustomBase.Code != 200 {
			t.Errorf("期望 Code 为 200，实际为 %d", resp.CustomBase.Code)
		}
		if resp.CustomBase.Output != grpcresult.DefaultSuccessOutput {
			t.Errorf("期望 Output 为 %s，实际为 %s", grpcresult.DefaultSuccessOutput, resp.CustomBase.Output)
		}
		if resp.CustomBase.Message != msg {
			t.Errorf("期望 Message 为 %q，实际为 %q", msg, resp.CustomBase.Message)
		}
		if resp.CustomBase.Context != "trace-beta-std-001" {
			t.Errorf("期望 Context 为 trace-beta-std-001，实际为 %s", resp.CustomBase.Context)
		}
		if resp.CustomBase.Overhead != nil {
			t.Errorf("非调试模式下期望 Overhead 为 nil，实际为 %v", resp.CustomBase.Overhead)
		}
		if resp.Data != "" {
			t.Errorf("未赋值的业务载荷 Data 期望为空字符串，实际为 %s", resp.Data)
		}

		// 真实 Protobuf 序列化与反序列化一致性验证
		rawBytes, err := proto.Marshal(resp)
		if err != nil {
			t.Fatalf("proto.Marshal 序列化失败: %v", err)
		}
		unmarshaled := &internaltest.ValidBetaResponse{}
		if err := proto.Unmarshal(rawBytes, unmarshaled); err != nil {
			t.Fatalf("proto.Unmarshal 反序列化失败: %v", err)
		}
		if !proto.Equal(resp, unmarshaled) {
			t.Errorf("反序列化后对象与原对象不一致: 原始=%v, 反序列化=%v", resp, unmarshaled)
		}
		if unmarshaled.CustomBase == nil || unmarshaled.CustomBase.Code != 200 || unmarshaled.CustomBase.Context != "trace-beta-std-001" {
			t.Errorf("反序列化后 CustomBase 关键字段校验失败: %v", unmarshaled.CustomBase)
		}
	})

	t.Run("调试模式自定义字段耗时注入", func(t *testing.T) {
		_ = os.Setenv(string(env.Debug), "true")
		defer func() { _ = os.Unsetenv(string(env.Debug)) }()

		startTime := time.Now().Add(-10 * time.Millisecond)
		ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "trace-beta-debug-002")
		ctx = context.WithValue(ctx, ctxkey.UserStartTimeKey, startTime)

		msg := "Beta with debug overhead"
		resp := grpcresult.SuccessWith[*internaltest.ValidBetaResponse](ctx, msg)
		if resp == nil || resp.CustomBase == nil {
			t.Fatal("期望返回非 nil 的响应与 CustomBase")
		}
		if resp.CustomBase.Context != "trace-beta-debug-002" {
			t.Errorf("期望 Context 为 trace-beta-debug-002，实际为 %s", resp.CustomBase.Context)
		}
		if resp.CustomBase.Overhead == nil || *resp.CustomBase.Overhead <= 0 {
			t.Errorf("调试模式下期望 Overhead 注入正整数耗时微秒，实际为 %v", resp.CustomBase.Overhead)
		}

		rawBytes, err := proto.Marshal(resp)
		if err != nil {
			t.Fatalf("proto.Marshal 序列化失败: %v", err)
		}
		unmarshaled := &internaltest.ValidBetaResponse{}
		if err := proto.Unmarshal(rawBytes, unmarshaled); err != nil {
			t.Fatalf("proto.Unmarshal 反序列化失败: %v", err)
		}
		if unmarshaled.CustomBase == nil || unmarshaled.CustomBase.Overhead == nil {
			t.Fatalf("反序列化后 CustomBase 或 Overhead 丢失: %v", unmarshaled)
		}
		if *unmarshaled.CustomBase.Overhead != *resp.CustomBase.Overhead {
			t.Errorf("反序列化后 Overhead 不一致: 期望 %d，实际 %d", *resp.CustomBase.Overhead, *unmarshaled.CustomBase.Overhead)
		}
	})
}

// TestSuccessWith_Panics_TypeConstraints 测试违反类型约束时的编程错误 panic。
func TestSuccessWith_Panics_TypeConstraints(t *testing.T) {
	ctx := context.Background()

	t.Run("非指针类型 int", func(t *testing.T) {
		assertPanic(t, "T must be a pointer type, got int", func() {
			_ = grpcresult.SuccessWith[int](ctx, "should panic")
		})
	})

	t.Run("非指针结构体 plainStruct", func(t *testing.T) {
		assertPanic(t, "T must be a pointer type, got struct", func() {
			_ = grpcresult.SuccessWith[plainStruct](ctx, "should panic")
		})
	})

	t.Run("指向非结构体的指针 *int", func(t *testing.T) {
		assertPanic(t, "T must be a pointer to struct, got pointer to int", func() {
			_ = grpcresult.SuccessWith[*int](ctx, "should panic")
		})
	})

	t.Run("未实现 proto.Message 的普通结构体指针 *nonProtoStruct", func(t *testing.T) {
		assertPanic(t, "does not implement proto.Message", func() {
			_ = grpcresult.SuccessWith[*nonProtoStruct](ctx, "should panic")
		})
	})

	t.Run("proto.Message 接口类型", func(t *testing.T) {
		assertPanic(t, "must be a concrete protobuf message pointer, got interface protoreflect.ProtoMessage", func() {
			_ = grpcresult.SuccessWith[proto.Message](ctx, "should panic")
		})
	})
}

// TestSuccessWith_Panics_FieldConstraints 测试 Protobuf 消息中 BaseResponse 字段缺失或重复时的编程错误 panic。
func TestSuccessWith_Panics_FieldConstraints(t *testing.T) {
	ctx := context.Background()

	t.Run("缺少 BaseResponse 字段的标准类型 wrapperspb.StringValue", func(t *testing.T) {
		assertPanic(t, "google.protobuf.StringValue has no field of type xBase.BaseResponse", func() {
			_ = grpcresult.SuccessWith[*wrapperspb.StringValue](ctx, "should panic")
		})
	})

	t.Run("缺少 BaseResponse 字段的 fixture 消息 MissingBaseResponse", func(t *testing.T) {
		assertPanic(t, "testdata.MissingBaseResponse has no field of type xBase.BaseResponse", func() {
			_ = grpcresult.SuccessWith[*internaltest.MissingBaseResponse](ctx, "should panic")
		})
	})

	t.Run("拥有多个 BaseResponse 字段的 fixture 消息 MultipleBaseResponse", func(t *testing.T) {
		assertPanic(t, "testdata.MultipleBaseResponse has 2 fields of type xBase.BaseResponse, expected exactly 1", func() {
			_ = grpcresult.SuccessWith[*internaltest.MultipleBaseResponse](ctx, "should panic")
		})
	})
}
