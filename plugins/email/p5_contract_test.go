package email

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/wneessen/go-mail"
)

// P5.2 强类型错误契约：12 个自有失败出口的包级签名断言。
//
// 断言以方法表达式形式固定签名，任何回退到原生 error 的改动都会在编译期失败。
var (
	_ func(*Plugin) *apperr.Error                                 = (*Plugin).acquire
	_ func(*Plugin, ...mail.Option) (*mail.Client, *apperr.Error) = (*Plugin).buildMailClient
	_ func(*Client, context.Context, *Message) *apperr.Error      = (*Client).Send
	_ func(*Client, context.Context, *Message) *apperr.Error      = (*Client).SendHTML
	_ func(*Client, context.Context, *Message) *apperr.Error      = (*Client).SendTemplate
	_ func(*Client, string) *apperr.Error                         = (*Client).AddTemplateDir
	_ func(*Client, fs.FS, string) *apperr.Error                  = (*Client).AddTemplateFS
	_ func(*Message, string) *apperr.Error                        = (*Message).AttachFile
	_ func(string) (*TemplateManager, *apperr.Error)              = newTemplateManager
	_ func(*TemplateManager, string, any) (string, *apperr.Error) = (*TemplateManager).Render
	_ func(*TemplateManager, string) *apperr.Error                = (*TemplateManager).AddDir
	_ func(*TemplateManager, fs.FS, string) *apperr.Error         = (*TemplateManager).AddFS
)

// W07 固定外部边界断言：watchedConn 保持原生 error 签名，
// 拨号回调保持 mail.DialContextFunc 的原生函数签名。
var (
	_ func(*watchedConn) error = (*watchedConn).Close
	_ mail.DialContextFunc     = func(context.Context, string, string) (net.Conn, error) { return nil, nil }
)

// failingCloseConn 是 Close 恒定失败的 net.Conn 替身，用于验证关闭失败原因保留。
type failingCloseConn struct {
	net.Conn
	closeErr error
}

// Close 返回预设的关闭错误。
func (c *failingCloseConn) Close() error { return c.closeErr }

// TestP5_WatchedConnCloseReturnsTrueNil 断言关闭成功时 Close 返回真正的 nil error，
// 且重复关闭保持幂等，不会把 nil 指针装入 error 接口冒充失败。
func TestP5_WatchedConnCloseReturnsTrueNil(t *testing.T) {
	_, peer := net.Pipe()
	t.Cleanup(func() { _ = peer.Close() })

	wc := &watchedConn{Conn: peer}
	if err := wc.Close(); err != nil {
		t.Fatalf("关闭成功时 Close 应返回真正的 nil，实际: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("重复关闭应保持幂等且返回真正的 nil，实际: %v", err)
	}
}

// TestP5_WatchedConnCloseKeepsFailureCause 断言关闭失败时保留底层原因，
// 且失败结果归一为所属域错误码 [EmailError]。
func TestP5_WatchedConnCloseKeepsFailureCause(t *testing.T) {
	boom := errors.New("close boom")
	wc := &watchedConn{Conn: &failingCloseConn{closeErr: boom}}

	err := wc.Close()
	if err == nil {
		t.Fatal("关闭失败时 Close 必须返回非 nil")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("关闭失败原因必须保留，实际: %v", err)
	}
	structured := apperr.Find(err)
	if structured == nil || structured.Code.Code != EmailError.Code {
		t.Fatalf("关闭失败应归一为 EmailError，实际: %v", err)
	}
}

// TestP5_SendCancelReturnsControlCanceled 断言发送期间上下文取消归一为 49900，
// 并保留原生 context.Canceled 哨兵。
func TestP5_SendCancelReturnsControlCanceled(t *testing.T) {
	srv := startFakeSMTPServer(t, true)

	_, factory := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))
	plugin := factory().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	t.Cleanup(func() { _ = plugin.Stop(context.Background()) })

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	msg := NewMessage()
	msg.To = []string{"cancel@test.local"}
	msg.Subject = "Cancel"
	msg.TextBody = "body"

	err := client.Send(ctx, msg)
	if err == nil {
		t.Fatal("取消后的发送必须失败")
	}
	if err.Code.Code != control.Canceled.Code {
		t.Fatalf("取消应归一为 %d，实际: %d (%v)", control.Canceled.Code, err.Code.Code, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("原生 context.Canceled 哨兵必须保留，实际: %v", err)
	}
}

// TestP5_SendTimeoutReturnsDeadlineExceeded 断言发送超时归一为 50401，
// 并保留原生 context.DeadlineExceeded 哨兵。
func TestP5_SendTimeoutReturnsDeadlineExceeded(t *testing.T) {
	srv := startFakeSMTPServer(t, true)

	_, factory := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))
	plugin := factory().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	t.Cleanup(func() { _ = plugin.Stop(context.Background()) })

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg := NewMessage()
	msg.To = []string{"timeout@test.local"}
	msg.Subject = "Timeout"
	msg.TextBody = "body"

	err := client.Send(ctx, msg)
	if err == nil {
		t.Fatal("超时后的发送必须失败")
	}
	if err.Code.Code != control.DeadlineExceeded.Code {
		t.Fatalf("超时应归一为 %d，实际: %d (%v)", control.DeadlineExceeded.Code, err.Code.Code, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("原生 context.DeadlineExceeded 哨兵必须保留，实际: %v", err)
	}
}

// TestP5_StartEmailPortKeepsDualCause 断言 EMAIL_PORT 解析失败时同时保留
// [ErrInvalidPort] 与底层值错误原因，且服务启动配置按所属 500 码归一。
func TestP5_StartEmailPortKeepsDualCause(t *testing.T) {
	t.Setenv("EMAIL_PORT", "not-a-number")

	_, factory := New(FromEnv())
	plugin := factory()

	err := plugin.Start(context.Background())
	if err == nil {
		t.Fatal("EMAIL_PORT 非法时 Start 必须失败")
	}
	if !errors.Is(err, ErrInvalidPort) {
		t.Fatalf("ErrInvalidPort 必须保留，实际: %v", err)
	}
	if err.Code.Code != EmailError.Code {
		t.Fatalf("服务启动配置应归一为 EmailError，实际: %d", err.Code.Code)
	}
	if !hasErrorCode(err, 40013) {
		t.Fatalf("底层值错误原因（40013）必须保留，实际: %v", err)
	}
}

// hasErrorCode 遍历错误链，判定是否存在指定错误码的统一错误。
func hasErrorCode(err error, code apperr.Code) bool {
	for cur := err; cur != nil; {
		if structured, ok := cur.(*apperr.Error); ok && structured.Code.Code == code {
			return true
		}
		switch wrapper := cur.(type) {
		case interface{ Unwrap() error }:
			cur = wrapper.Unwrap()
		case interface{ Unwrap() []error }:
			for _, child := range wrapper.Unwrap() {
				if hasErrorCode(child, code) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

// TestP5_PasswordNeverLeaks 断言失败结果既不在错误文本中、也不在 JSON 序列化中
// 泄漏 SMTP 密码。
func TestP5_PasswordNeverLeaks(t *testing.T) {
	const secret = "sup3r-secret-pass"

	_, factory := New(WithConfig(Config{
		Host:     "127.0.0.1",
		Port:     1,
		Username: "u",
		Password: secret,
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))
	plugin := factory().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	t.Cleanup(func() { _ = plugin.Stop(context.Background()) })

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	msg := NewMessage()
	msg.To = []string{"leak@test.local"}
	msg.Subject = "Leak"
	msg.TextBody = "body"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Send(ctx, msg)
	if err == nil {
		t.Fatal("指向不可用端口的发送必须失败")
	}
	if containsSecret(err.Error(), secret) {
		t.Fatalf("错误文本泄漏密码: %v", err)
	}
	if structured := apperr.Find(err); structured != nil {
		if containsSecret(string(structured.ErrorMessage), secret) {
			t.Fatalf("ErrorMessage 泄漏密码: %v", structured.ErrorMessage)
		}
	}
}

// containsSecret 判定文本是否包含敏感串。
func containsSecret(text, secret string) bool {
	return len(secret) > 0 && len(text) >= len(secret) && indexOf(text, secret) >= 0
}

// indexOf 返回子串首次出现的位置，未出现返回 -1。
func indexOf(text, sub string) int {
	for i := 0; i+len(sub) <= len(text); i++ {
		if text[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestP5_NilReceiverSafety 断言各自有出口在 nil 接收器下安全返回强类型失败，
// 不发生 panic。
func TestP5_NilReceiverSafety(t *testing.T) {
	var (
		nilClient  *Client
		nilMessage *Message
		nilManager *TemplateManager
		nilPlugin  *Plugin
	)

	if err := nilClient.Send(context.Background(), NewMessage()); err == nil {
		t.Fatal("nil Client 的 Send 必须返回失败")
	}
	if err := nilClient.SendHTML(context.Background(), NewMessage()); err == nil {
		t.Fatal("nil Client 的 SendHTML 必须返回失败")
	}
	if err := nilClient.SendTemplate(context.Background(), NewMessage()); err == nil {
		t.Fatal("nil Client 的 SendTemplate 必须返回失败")
	}
	if err := nilClient.AddTemplateDir("dir"); err == nil {
		t.Fatal("nil Client 的 AddTemplateDir 必须返回失败")
	}
	if err := nilClient.AddTemplateFS(nil, "*.html"); err == nil {
		t.Fatal("nil Client 的 AddTemplateFS 必须返回失败")
	}
	if err := nilMessage.AttachFile("file.txt"); err == nil {
		t.Fatal("nil Message 的 AttachFile 必须返回失败")
	}
	if _, err := nilManager.Render("welcome", nil); err == nil {
		t.Fatal("nil TemplateManager 的 Render 必须返回失败")
	}
	if err := nilManager.AddDir("dir"); err == nil {
		t.Fatal("nil TemplateManager 的 AddDir 必须返回失败")
	}
	if err := nilManager.AddFS(nil, "*.html"); err == nil {
		t.Fatal("nil TemplateManager 的 AddFS 必须返回失败")
	}
	if err := nilPlugin.acquire(); err == nil {
		t.Fatal("nil Plugin 的 acquire 必须返回失败")
	}
	if _, err := nilPlugin.buildMailClient(); err == nil {
		t.Fatal("nil Plugin 的 buildMailClient 必须返回失败")
	}
}

// TestP5_TemplateManagerStrongErrors 断言模板管理器的失败出口归一为 EmailError。
func TestP5_TemplateManagerStrongErrors(t *testing.T) {
	manager, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}
	if err != nil {
		t.Fatal("成功路径必须返回真正的 nil 指针")
	}

	if _, renderErr := manager.Render("not_exists", nil); renderErr == nil {
		t.Fatal("模板不存在时必须失败")
	} else if renderErr.Code.Code != EmailError.Code {
		t.Fatalf("渲染失败应归一为 EmailError，实际: %d", renderErr.Code.Code)
	}

	if dirErr := manager.AddDir("/definitely/not/exist/dir"); dirErr == nil {
		t.Fatal("目录不存在时必须失败")
	} else if dirErr.Code.Code != EmailError.Code {
		t.Fatalf("AddDir 失败应归一为 EmailError，实际: %d", dirErr.Code.Code)
	}

	if fsErr := manager.AddFS(nil, "*.html"); fsErr == nil {
		t.Fatal("文件系统为空时必须失败")
	} else {
		if fsErr.Code.Code != EmailError.Code {
			t.Fatalf("AddFS 失败应归一为 EmailError，实际: %d", fsErr.Code.Code)
		}
		if errors.Is(fsErr, ErrNotInitialized) {
			t.Fatal("AddFS 参数为空不得误绑 ErrNotInitialized 哨兵")
		}
	}

	if patErr := manager.AddFS(defaultTemplateFS, ""); patErr == nil {
		t.Fatal("匹配模式为空时必须失败")
	} else {
		if patErr.Code.Code != EmailError.Code {
			t.Fatalf("AddFS 空模式应归一为 EmailError，实际: %d", patErr.Code.Code)
		}
		if errors.Is(patErr, ErrNotInitialized) {
			t.Fatal("AddFS 空模式不得误绑 ErrNotInitialized 哨兵")
		}
	}

	zeroManager := &TemplateManager{}
	if err := zeroManager.AddDir(t.TempDir()); err == nil || !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("零值 TemplateManager.AddDir 必须返回 ErrNotInitialized 且不 panic，实际: %v", err)
	}
	if err := zeroManager.AddFS(defaultTemplateFS, "template/*.html"); err == nil || !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("零值 TemplateManager.AddFS 必须返回 ErrNotInitialized 且不 panic，实际: %v", err)
	}
}

type customErrContext struct {
	context.Context
	err error
}

func (c customErrContext) Err() error { return c.err }

// TestP5_OpErrWithContext_CustomContextErrorDoesNotPanic 断言非标准 ctx.Err() 不会触发空指针 panic。
func TestP5_OpErrWithContext_CustomContextErrorDoesNotPanic(t *testing.T) {
	ctxErr := errors.New("custom context failure")
	cause := errors.New("smtp transport broken")
	ctx := customErrContext{Context: context.Background(), err: ctxErr}

	got := opErrWithContext("email: 发送失败", cause, ctx)
	if got == nil {
		t.Fatal("期望返回非 nil 错误")
	}
	if got.Code.Code != EmailError.Code {
		t.Fatalf("非标准 ctx.Err() 应回退 EmailError，实际: %d", got.Code.Code)
	}
	if !errors.Is(got, ctxErr) || !errors.Is(got, cause) {
		t.Fatalf("必须同时保留 ctxErr 与 cause，实际: %v", got)
	}
}
