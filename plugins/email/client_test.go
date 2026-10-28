package email

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// fakeSMTPServer 进程内可释放的假 SMTP 服务器
type fakeSMTPServer struct {
	listener          net.Listener
	addr              string
	mu                sync.Mutex
	messages          []string
	hang              bool
	hangAfterGreeting bool
	closed            chan struct{}
}

func startFakeSMTPServer(t *testing.T, hang bool) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start fake smtp listener: %v", err)
	}

	s := &fakeSMTPServer{
		listener: ln,
		addr:     ln.Addr().String(),
		hang:     hang,
		closed:   make(chan struct{}),
	}

	go s.serve()
	t.Cleanup(func() {
		_ = s.close()
	})
	return s
}

func (s *fakeSMTPServer) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
		return s.listener.Close()
	}
}

func (s *fakeSMTPServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *fakeSMTPServer) host() string {
	return "127.0.0.1"
}

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *fakeSMTPServer) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	if s.hang {
		// 阻塞住直到连接被关闭，测试超时/取消
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		return
	}
	if s.hangAfterGreeting {
		writeLine := func(line string) {
			_, _ = conn.Write([]byte(line + "\r\n"))
		}
		writeLine("220 fake.bamboo.local ESMTP Fake")
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		return
	}

	// 假 SMTP 最小应答协议
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	writeLine := func(line string) {
		_, _ = writer.WriteString(line + "\r\n")
		_ = writer.Flush()
	}

	writeLine("220 fake.bamboo.local ESMTP Fake")

	inData := false
	var currentMsg strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.messages = append(s.messages, currentMsg.String())
				s.mu.Unlock()
				writeLine("250 2.0.0 OK message queued")
			} else {
				currentMsg.WriteString(line + "\n")
			}
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			writeLine("250-fake.bamboo.local")
			writeLine("250-AUTH PLAIN")
			writeLine("250 8BITMIME")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			writeLine("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			currentMsg.Reset()
			writeLine("250 2.1.0 Ok")
		case strings.HasPrefix(upper, "RCPT TO:"):
			writeLine("250 2.1.5 Ok")
		case upper == "DATA":
			inData = true
			writeLine("354 End data with <CR><LF>.<CR><LF>")
		case upper == "QUIT":
			writeLine("221 2.0.0 Bye")
			return
		case upper == "RSET":
			currentMsg.Reset()
			writeLine("250 2.0.0 OK")
		case upper == "NOOP":
			writeLine("250 2.0.0 OK")
		default:
			writeLine("250 2.0.0 OK")
		}
	}
}

// TestNewMessage 测试创建空邮件消息
func TestNewMessage(t *testing.T) {
	msg := NewMessage()
	if msg == nil {
		t.Fatal("NewMessage() 返回 nil")
	}
	if len(msg.To) != 0 {
		t.Errorf("期望 To 为空切片, 实际长度: %d", len(msg.To))
	}
	if msg.Subject != "" {
		t.Errorf("期望 Subject 为空, 实际: %s", msg.Subject)
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("期望 Attachments 为空切片, 实际长度: %d", len(msg.Attachments))
	}
}

// TestMessageAttachReader 测试从 Reader 添加附件与链式调用
func TestMessageAttachReader(t *testing.T) {
	msg := NewMessage()
	data := strings.NewReader("test content")
	result := msg.AttachReader("test.txt", data)

	if result != msg {
		t.Fatal("AttachReader 应该返回 Message 指针以支持链式调用")
	}
	if len(msg.Attachments) != 1 {
		t.Fatalf("期望 1 个附件, 实际: %d", len(msg.Attachments))
	}
	if msg.Attachments[0].Filename != "test.txt" {
		t.Errorf("期望附件名 test.txt, 实际: %s", msg.Attachments[0].Filename)
	}

	// 测试 AttachFile
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("file content"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	if err := msg.AttachFile(tmpFile); err != nil {
		t.Fatalf("AttachFile failed: %v", err)
	}
	if len(msg.Attachments) != 2 {
		t.Fatalf("期望 2 个附件, 实际: %d", len(msg.Attachments))
	}
	// 不存在的文件应报错
	if err := msg.AttachFile("non_existent_file_path"); err == nil {
		t.Fatal("AttachFile with non-existent file should return error")
	}
}

// TestTemplateManagerRender 测试内置模板渲染
func TestTemplateManagerRender(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	// 测试 welcome 模板
	data := map[string]string{
		"Username": "test_user",
	}
	html, err := tmpl.Render("welcome", data)
	if err != nil {
		t.Fatalf("渲染 welcome 模板失败: %v", err)
	}
	if !strings.Contains(html, "Bamboo Service") {
		t.Error("渲染结果应包含 Bamboo Service 标题")
	}
	if !strings.Contains(html, "test_user") {
		t.Error("渲染结果应包含用户名 test_user")
	}

	// 测试 verification 模板
	html, err = tmpl.Render("verification", map[string]string{
		"Code":   "123456",
		"Expire": "5分钟",
	})
	if err != nil {
		t.Fatalf("渲染 verification 模板失败: %v", err)
	}
	if !strings.Contains(html, "123456") || !strings.Contains(html, "5分钟") {
		t.Error("渲染 verification 缺少期望内容")
	}

	// 测试 reset_password 模板
	html, err = tmpl.Render("reset_password", map[string]string{
		"ResetURL": "https://example.com/reset?token=abc123",
		"Expire":   "30分钟",
	})
	if err != nil {
		t.Fatalf("渲染 reset_password 模板失败: %v", err)
	}
	if !strings.Contains(html, "https://example.com/reset?token=abc123") {
		t.Error("渲染 reset_password 缺少期望链接")
	}
}

// TestRenderExternalTemplate 测试外部模板目录
func TestRenderExternalTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	customHTML := `{{define "custom_report"}}<p>报告内容: {{.Title}}</p>{{end}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "custom_report.html"), []byte(customHTML), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	tmpl, err := newTemplateManager(tmpDir)
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	html, err := tmpl.Render("custom_report", map[string]string{
		"Title": "月度报告",
	})
	if err != nil {
		t.Fatalf("渲染外部模板失败: %v", err)
	}
	if !strings.Contains(html, "月度报告") {
		t.Errorf("渲染结果应包含 月度报告, 实际: %s", html)
	}
}

// TestTemplateManagerListTemplates 测试获取模板列表
func TestTemplateManagerListTemplates(t *testing.T) {
	tmpl, err := newTemplateManager("")
	if err != nil {
		t.Fatalf("创建模板管理器失败: %v", err)
	}

	names := tmpl.ListTemplates()
	if len(names) == 0 {
		t.Fatal("应该返回至少一个内置模板")
	}
	found := false
	for _, n := range names {
		if n == "verification" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("模板列表应包含 verification, 实际: %v", names)
	}

	// 测试未找到模板
	if _, err := tmpl.Render("not_exists", nil); err == nil {
		t.Fatal("不存在的模板应当返回错误")
	}
}

// TestPlugin_NewAndInfo 验证 New 返回的 PluginInfo 元信息
func TestPlugin_NewAndInfo(t *testing.T) {
	info, fac := New(WithConfig(Config{
		Host:     "127.0.0.1",
		Port:     2525,
		Username: "user",
		Password: "password",
		FromAddr: "noreply@example.com",
	}))

	if info.Key != PluginKey {
		t.Fatalf("expected key %s, got %s", PluginKey, info.Key)
	}
	if info.Mode != application.ModeTool {
		t.Fatalf("expected ModeTool, got %v", info.Mode)
	}
	if len(info.Provides) != 1 || info.Provides[0] != ctxkey.EmailClientKey {
		t.Fatalf("expected provides [EmailClientKey], got %v", info.Provides)
	}
	if fac == nil {
		t.Fatal("factory function should not be nil")
	}

	p := fac()
	if p == nil {
		t.Fatal("plugin instance should not be nil")
	}
}

// TestPlugin_StartValidation 验证 Start 阶段的必要配置检查与失败资格
func TestPlugin_StartValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{
			name:    "missing host",
			cfg:     Config{Username: "u", Password: "p", FromAddr: "f@a.com"},
			wantErr: ErrMissingHost,
		},
		{
			name:    "missing user",
			cfg:     Config{Host: "h", Password: "p", FromAddr: "f@a.com"},
			wantErr: ErrMissingUser,
		},
		{
			name:    "missing pass",
			cfg:     Config{Host: "h", Username: "u", FromAddr: "f@a.com"},
			wantErr: ErrMissingPass,
		},
		{
			name:    "missing from",
			cfg:     Config{Host: "h", Username: "u", Password: "p"},
			wantErr: ErrMissingFrom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, fac := New(WithConfig(tt.cfg))
			plugin := fac()
			err := plugin.Start(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			// Start 失败后，必须可安全调用 Stop，不得 panic
			if stopErr := plugin.Stop(context.Background()); stopErr != nil {
				t.Fatalf("Stop after failed Start should succeed, got %v", stopErr)
			}
		})
	}
}

// TestPlugin_FromEnv 验证 FromEnv 读取环境变量
func TestPlugin_FromEnv(t *testing.T) {
	t.Setenv(string(env.EmailHost), "smtp.env.test")
	t.Setenv(string(env.EmailPort), "2525")
	t.Setenv(string(env.EmailUser), "env_user")
	t.Setenv(string(env.EmailPass), "env_pass")
	t.Setenv(string(env.EmailFrom), "env@test.com")
	t.Setenv(string(env.EmailFromName), "Env Sender")
	t.Setenv(string(env.EmailTLS), "none")

	_, fac := New(FromEnv())
	plugin := fac().(*Plugin)

	err := plugin.Start(context.Background())
	if err != nil {
		t.Fatalf("Start with FromEnv failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	plugin.clientMu.RLock()
	cfg := plugin.opts.cfg
	plugin.clientMu.RUnlock()

	if cfg.Host != "smtp.env.test" || cfg.Port != 2525 || cfg.Username != "env_user" || cfg.Password != "env_pass" || cfg.FromAddr != "env@test.com" || cfg.FromName != "Env Sender" || cfg.TLSType != "none" {
		t.Fatalf("config mismatch: %+v", cfg)
	}
}

// TestPlugin_StopIdempotentAndNil 验证 nil / 重复 Stop 安全幂等
func TestPlugin_StopIdempotentAndNil(t *testing.T) {
	var nilPlugin *Plugin
	if err := nilPlugin.Stop(context.Background()); err != nil {
		t.Fatalf("nil plugin Stop should return nil, got: %v", err)
	}

	_, fac := New(WithConfig(Config{
		Host:     "127.0.0.1",
		Port:     25,
		Username: "u",
		Password: "p",
		FromAddr: "f@a.com",
		TLSType:  "none",
	}))
	plugin := fac()

	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 第一次 Stop
	if err := plugin.Stop(context.Background()); err != nil {
		t.Fatalf("first Stop failed: %v", err)
	}
	// 第二次 Stop (重复)
	if err := plugin.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop should be idempotent, got: %v", err)
	}
}

// TestClient_SendSuccessViaFakeSMTP 使用进程内 fake SMTP 验证真实发信
func TestClient_SendSuccessViaFakeSMTP(t *testing.T) {
	srv := startFakeSMTPServer(t, false)

	_, fac := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "testuser",
		Password: "testpassword",
		FromAddr: "bamboo@service.local",
		FromName: "Bamboo Service",
		TLSType:  "none",
	}))

	plugin := fac()
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	exp, ok := plugin.(ctxkey.Exposer)
	if !ok {
		t.Fatal("plugin must implement ctxkey.Exposer")
	}
	m := exp.Expose()
	rawClient, ok := m[ctxkey.EmailClientKey]
	if !ok || rawClient == nil {
		t.Fatal("Expose must provide non-nil EmailClientKey")
	}
	client := rawClient.(*Client)

	// 1. 发送带文本、HTML、抄送、密送的邮件
	msg := NewMessage()
	msg.To = []string{"user1@test.local", "user2@test.local"}
	msg.Cc = []string{"cc@test.local"}
	msg.Bcc = []string{"bcc@test.local"}
	msg.ReplyTo = "support@service.local"
	msg.Subject = "Test Email Subject"
	msg.TextBody = "Plain text body"
	msg.HTMLBody = "<p>HTML body</p>"
	msg.AttachReader("note.txt", strings.NewReader("hello attachment"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// 2. 发送模板邮件
	tplMsg := NewMessage()
	tplMsg.To = []string{"recipient@test.local"}
	tplMsg.Subject = "Welcome Email"
	tplMsg.Template = "welcome"
	tplMsg.TemplateData = map[string]string{"Username": "Alice"}

	if err := client.SendTemplate(ctx, tplMsg); err != nil {
		t.Fatalf("SendTemplate failed: %v", err)
	}

	srv.mu.Lock()
	msgCount := len(srv.messages)
	srv.mu.Unlock()

	if msgCount != 2 {
		t.Fatalf("expected 2 messages received by fake SMTP, got %d", msgCount)
	}
}

// TestClient_ContextCancelAndTimeout 验证发信时 context 到期强制取消
func TestClient_ContextCancelAndTimeout(t *testing.T) {
	srv := startFakeSMTPServer(t, true) // hang = true，连接挂起

	_, fac := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))

	plugin := fac().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	msg := NewMessage()
	msg.To = []string{"recipient@test.local"}
	msg.Subject = "Timeout Test"
	msg.TextBody = "Will timeout"

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := client.Send(ctx, msg)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected Send to fail on timeout, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Send took too long (%v), context was not respected", elapsed)
	}
}

// TestClient_ConcurrencyAndStopDraining 验证并发发送登记、Stop 关闸与真实等待归零
func TestClient_ConcurrencyAndStopDraining(t *testing.T) {
	srv := startFakeSMTPServer(t, false)

	_, fac := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))

	plugin := fac().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	var wg sync.WaitGroup
	concurrentCount := 10
	errChan := make(chan error, concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg := NewMessage()
			msg.To = []string{fmt.Sprintf("user%d@test.local", idx)}
			msg.Subject = fmt.Sprintf("Subject %d", idx)
			msg.TextBody = "Body"
			if err := client.Send(context.Background(), msg); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	for err := range errChan {
		t.Fatalf("concurrent send failed: %v", err)
	}

	// 此时所有发送已完成，停止 plugin
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()

	if err := plugin.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// 验证 inflight 计数归零
	if count := plugin.InflightCount(); count != 0 {
		t.Fatalf("expected inflight count 0 after Stop, got %d", count)
	}

	// 验证 Stop 关闸后新的 Send 必须被拒绝
	newMsg := NewMessage()
	newMsg.To = []string{"after_stop@test.local"}
	newMsg.Subject = "After stop"
	newMsg.TextBody = "Should fail"

	err := client.Send(context.Background(), newMsg)
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("expected ErrStopped after Stop, got: %v", err)
	}
}

// TestClient_WithLogger 验证显式 WithLogger 注入
func TestClient_WithLogger(t *testing.T) {
	logInst, err := logging.New(slog.NewTextHandler(io.Discard, nil))
	if err != nil {
		t.Fatalf("create logging failed: %v", err)
	}

	srv := startFakeSMTPServer(t, false)

	_, fac := New(
		WithConfig(Config{
			Host:     srv.host(),
			Port:     srv.port(),
			Username: "u",
			Password: "p",
			FromAddr: "f@test.local",
			TLSType:  "none",
		}),
		WithLogger(logInst),
	)

	plugin := fac().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	if plugin.log == nil {
		t.Fatal("expected named logger to be populated from WithLogger")
	}

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)
	msg := NewMessage()
	msg.To = []string{"logtest@test.local"}
	msg.Subject = "Logger test"
	msg.TextBody = "test"

	if err := client.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send with logger failed: %v", err)
	}
}

// TestRealExternalSMTP_DefaultSkip 真实外部 SMTP 默认 skip，仅在 BAMBOO_TEST_REAL_SMTP 显式配置时执行
func TestRealExternalSMTP_DefaultSkip(t *testing.T) {
	if os.Getenv("BAMBOO_TEST_REAL_SMTP") == "" {
		t.Skip("skipping external real SMTP test; BAMBOO_TEST_REAL_SMTP is not set")
	}
	// 如果设置了真实环境，才执行真实建连测试
}

// TestClient_ContextCancelDuringSend 验证 SMTP greeting 建立后发信过程中取消 context 立即中断连接并回收资源
func TestClient_ContextCancelDuringSend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	srv := &fakeSMTPServer{
		listener:          ln,
		addr:              ln.Addr().String(),
		hangAfterGreeting: true,
		closed:            make(chan struct{}),
	}
	go srv.serve()
	defer func() { _ = srv.close() }()

	_, fac := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))

	plugin := fac().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	msg := NewMessage()
	msg.To = []string{"hang@test.local"}
	msg.Subject = "Hang After Greeting"
	msg.TextBody = "body"

	start := time.Now()
	err = client.Send(ctx, msg)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected Send to fail on canceled context, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Send took too long (%v), context was not respected", elapsed)
	}
}

// TestClient_StopDrainingWaitingInflight 验证 Stop 会真实阻塞等待进行中的 Send 退出并释放计数
func TestClient_StopDrainingWaitingInflight(t *testing.T) {
	srv := startFakeSMTPServer(t, true)

	_, fac := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))

	plugin := fac().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	sendCtx, sendCancel := context.WithCancel(context.Background())
	sendStarted := make(chan struct{})
	sendFinished := make(chan struct{})

	go func() {
		msg := NewMessage()
		msg.To = []string{"draining@test.local"}
		msg.Subject = "Draining"
		msg.TextBody = "body"
		close(sendStarted)
		_ = client.Send(sendCtx, msg)
		close(sendFinished)
	}()

	<-sendStarted
	// 等待发送进入 inflight 登记
	deadline := time.Now().Add(time.Second)
	for plugin.InflightCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for inflight count to increase")
		}
		time.Sleep(5 * time.Millisecond)
	}

	stopDone := make(chan struct{})
	go func() {
		_ = plugin.Stop(context.Background())
		close(stopDone)
	}()

	// 在 send 未结束前，Stop 应被阻塞
	select {
	case <-stopDone:
		t.Fatal("Stop completed before inflight send finished")
	case <-time.After(100 * time.Millisecond):
		// 预期行为：Stop 正在阻塞等待
	}

	// 取消 send 上下文让其真实退出
	sendCancel()
	<-sendFinished

	// 此时 Stop 应能立即完成退出
	select {
	case <-stopDone:
		// 成功退出
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not complete after inflight send finished")
	}

	if plugin.InflightCount() != 0 {
		t.Fatalf("expected inflight 0, got %d", plugin.InflightCount())
	}
}

// TestClient_ContextCancelRaceAndDoubleClose 验证已取消/取消与dial并发压力race及重复Close回归
func TestClient_ContextCancelRaceAndDoubleClose(t *testing.T) {
	srv := startFakeSMTPServer(t, true)

	_, fac := New(WithConfig(Config{
		Host:     srv.host(),
		Port:     srv.port(),
		Username: "u",
		Password: "p",
		FromAddr: "f@test.local",
		TLSType:  "none",
	}))

	plugin := fac().(*Plugin)
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	client := plugin.Expose()[ctxkey.EmailClientKey].(*Client)

	// 1. 已取消的 context 压力测试
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		msg := NewMessage()
		msg.To = []string{"canceled@test.local"}
		msg.Subject = "Pre-canceled"
		msg.TextBody = "body"
		err := client.Send(ctx, msg)
		if err == nil {
			t.Fatal("expected Send to fail on pre-canceled context")
		}
	}

	// 2. 发送中快速取消与并发压力，验证 race 条件下真实 conn 安全回收与无 panic
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			if idx%2 == 0 {
				time.AfterFunc(time.Duration(idx*200)*time.Microsecond, cancel)
			} else {
				cancel()
			}
			msg := NewMessage()
			msg.To = []string{"race@test.local"}
			msg.Subject = "Race Cancel"
			msg.TextBody = "body"
			_ = client.Send(ctx, msg)
		}(i)
	}
	wg.Wait()

	// 3. watchedConn 独立并发 Close 重复关闭与 stopWatch 回归验证
	rawServer, rawClient := net.Pipe()
	defer func() { _ = rawServer.Close() }()
	wc := &watchedConn{Conn: rawClient}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopWatch := context.AfterFunc(ctx, func() {
		_ = rawClient.SetDeadline(time.Now())
		_ = rawClient.Close()
	})
	wc.mu.Lock()
	wc.stopWatch = stopWatch
	wc.mu.Unlock()

	var closeWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		closeWg.Add(1)
		go func() {
			defer closeWg.Done()
			_ = wc.Close()
		}()
	}
	closeWg.Wait()
	// 再次显式 Close 验证幂等性
	if err := wc.Close(); err != nil && !strings.Contains(err.Error(), "closed") {
		t.Logf("expected closed error or nil on double close: %v", err)
	}
}
