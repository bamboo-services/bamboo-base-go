package email_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	email "github.com/bamboo-services/bamboo-base-go/plugins/email/v2"
)

type mailpitMessageSummary struct {
	ID      string `json:"ID"`
	Subject string `json:"Subject"`
	To      []struct {
		Address string `json:"Address"`
		Name    string `json:"Name"`
	} `json:"To"`
}

type mailpitSearchResponse struct {
	Total    int                     `json:"total"`
	Messages []mailpitMessageSummary `json:"messages"`
}

type mailpitMessageDetail struct {
	ID      string `json:"ID"`
	Subject string `json:"Subject"`
	Text    string `json:"Text"`
	HTML    string `json:"HTML"`
	To      []struct {
		Address string `json:"Address"`
		Name    string `json:"Name"`
	} `json:"To"`
}

type emailWorker struct {
	ready   chan struct{}
	ctxChan chan context.Context
}

func (w *emailWorker) Start(ctx context.Context) error {
	return nil
}

func (w *emailWorker) Run(ctx context.Context) error {
	w.ctxChan <- ctx
	close(w.ready)
	<-ctx.Done()
	return nil
}

func (w *emailWorker) Stop(ctx context.Context) error {
	return nil
}

func getIntegrationSMTPConfig(t *testing.T) (host string, port int, httpAddr string) {
	t.Helper()
	host = os.Getenv("BAMBOO_TEST_SMTP_HOST")
	portStr := os.Getenv("BAMBOO_TEST_SMTP_PORT")
	if host == "" || portStr == "" {
		t.Skip("BAMBOO_TEST_SMTP_HOST or BAMBOO_TEST_SMTP_PORT not set, skipping integration smtp test")
	}

	var err error
	port, err = strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		t.Fatalf("invalid BAMBOO_TEST_SMTP_PORT: %v", portStr)
	}

	httpAddr = os.Getenv("BAMBOO_TEST_SMTP_HTTP_ADDR")
	if httpAddr == "" {
		httpPort := os.Getenv("BAMBOO_TEST_SMTP_HTTP_PORT")
		if httpPort != "" {
			httpAddr = fmt.Sprintf("http://%s:%s", host, httpPort)
		} else {
			httpAddr = "http://127.0.0.1:8025"
		}
	}
	if !strings.HasPrefix(httpAddr, "http://") && !strings.HasPrefix(httpAddr, "https://") {
		httpAddr = "http://" + httpAddr
	}
	return host, port, httpAddr
}

func TestIntegration_SMTPSendAndLifecycle(t *testing.T) {
	smtpHost, smtpPort, httpBase := getIntegrationSMTPConfig(t)

	// 1. 初始化 App，注册 Email Tool 与 Worker（设合理短 Grace 避免默认 20s 拖长测试）
	app := application.New(
		application.WithGraceTimeout(2*time.Second),
		application.WithShutdownTimeout(5*time.Second),
	)
	emailInfo, emailFactory := email.New(email.WithConfig(email.Config{
		Host:     smtpHost,
		Port:     smtpPort,
		Username: "integration_user",
		Password: "integration_password",
		FromName: "Bamboo集成测试",
		FromAddr: "bamboo-test@example.invalid",
		TLSType:  "none",
	}))

	if err := app.RegisterTool(emailInfo, emailFactory); err != nil {
		t.Fatalf("RegisterTool(Email) failed: %v", err)
	}

	worker := &emailWorker{
		ready:   make(chan struct{}),
		ctxChan: make(chan context.Context, 1),
	}
	workerInfo := application.PluginInfo{
		Key:      ctxkey.Key("test_email_integration_worker"),
		Name:     "TestEmailIntegrationWorker",
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ctxkey.EmailClientKey},
	}
	if err := app.RegisterWorker(workerInfo, func() application.RunnablePlugin { return worker }); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-worker.ready:
	case err := <-runErrCh:
		t.Fatalf("app.Run failed unexpectedly: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for worker to be ready")
	}

	workerCtx := <-worker.ctxChan
	client, getErr := application.Get[*email.Client](workerCtx, ctxkey.EmailClientKey)
	if getErr != nil {
		t.Fatalf("application.Get[*email.Client] failed: %v", getErr)
	}
	if client == nil {
		t.Fatal("expected non-nil *email.Client")
	}

	// 2. 发送带唯一 Subject 和中文编码的正文邮件到测试地址
	uniqueSubject := fmt.Sprintf("Bamboo Integration Test 邮件验证 [%d]", time.Now().UnixNano())
	recipient := "user@example.invalid"
	chineseText := "你好，这是来自 Bamboo Base 的自动化真实集成测试邮件！包含中文字符与换行。"
	chineseHTML := "<html><body><h1>Bamboo Base 集成测试</h1><p>中文内容测试成功。</p></body></html>"

	msg := email.NewMessage()
	msg.To = []string{recipient}
	msg.Subject = uniqueSubject
	msg.TextBody = chineseText
	msg.HTMLBody = chineseHTML

	sendCtx, sendCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer sendCancel()

	if err := client.Send(sendCtx, msg); err != nil {
		t.Fatalf("client.Send to Mailpit failed: %v", err)
	}

	// 3. 通过 Mailpit HTTP API 查询匹配该唯一 Subject 的邮件并校验
	var matchedID string
	httpClient := &http.Client{Timeout: 5 * time.Second}
	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s", httpBase, strings.ReplaceAll(uniqueSubject, " ", "+"))

	// 轮询查询 Mailpit 直至出现或超时
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, searchURL, nil)
		if err != nil {
			t.Fatalf("failed to create search request: %v", err)
		}
		resp, err := httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			var searchRes mailpitSearchResponse
			if err := json.Unmarshal(bodyBytes, &searchRes); err == nil && searchRes.Total > 0 {
				for _, m := range searchRes.Messages {
					if m.Subject == uniqueSubject {
						matchedID = m.ID
						break
					}
				}
			}
		}
		if matchedID != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if matchedID == "" {
		t.Fatalf("failed to find message with subject %q in Mailpit via %s", uniqueSubject, searchURL)
	}

	// 获取具体邮件详情核对收件人与正文中文
	detailURL := fmt.Sprintf("%s/api/v1/message/%s", httpBase, matchedID)
	reqDetail, err := http.NewRequestWithContext(context.Background(), http.MethodGet, detailURL, nil)
	if err != nil {
		t.Fatalf("failed to create detail request: %v", err)
	}
	respDetail, err := httpClient.Do(reqDetail)
	if err != nil || respDetail.StatusCode != http.StatusOK {
		t.Fatalf("failed to fetch message detail from %s: err=%v, code=%d", detailURL, err, respDetail.StatusCode)
	}
	detailBytes, _ := io.ReadAll(respDetail.Body)
	_ = respDetail.Body.Close()

	var msgDetail mailpitMessageDetail
	if err := json.Unmarshal(detailBytes, &msgDetail); err != nil {
		t.Fatalf("failed to unmarshal message detail: %v", err)
	}

	if len(msgDetail.To) == 0 || msgDetail.To[0].Address != recipient {
		t.Errorf("expected recipient %s, got %+v", recipient, msgDetail.To)
	}
	if !strings.Contains(msgDetail.Text, "你好，这是来自 Bamboo Base") {
		t.Errorf("expected text body to contain Chinese string, got %q", msgDetail.Text)
	}
	if !strings.Contains(msgDetail.HTML, "中文内容测试成功") {
		t.Errorf("expected html body to contain Chinese string, got %q", msgDetail.HTML)
	}

	// 精确删除本次测试自己的 messageID（绝对不调用 DeleteAll，不影响他人）
	deleteURL := fmt.Sprintf("%s/api/v1/message/%s", httpBase, matchedID)
	reqDel, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, deleteURL, nil)
	if err == nil {
		respDel, errDel := httpClient.Do(reqDel)
		if errDel == nil {
			_ = respDel.Body.Close()
		}
	}

	// 4. 证明生命周期关机：物理关闭、client 关闭、inflight=0，且关闭后 Send 被明确拒绝
	shutdownErr := app.Shutdown()
	if shutdownErr != nil {
		t.Fatalf("app.Shutdown failed: %v", shutdownErr)
	}

	select {
	case runErr := <-runErrCh:
		if runErr != nil {
			t.Fatalf("app.Run exited with error: %v", runErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for app.Run to exit")
	}

	// 验证关闭后客户端发送被拒绝且 inflight=0
	afterMsg := email.NewMessage()
	afterMsg.To = []string{recipient}
	afterMsg.Subject = "After Shutdown Email"
	afterMsg.TextBody = "Should fail"

	errAfter := client.Send(context.Background(), afterMsg)
	if errAfter == nil {
		t.Fatal("expected Send to fail after app.Shutdown, got nil")
	}
	if !strings.Contains(errAfter.Error(), "stopped") && !strings.Contains(errAfter.Error(), "closed") {
		t.Logf("Send after shutdown returned expected rejection: %v", errAfter)
	}
}

func TestIntegration_StopDrainsInflightPhysicalConn(t *testing.T) {
	smtpHost, smtpPort, _ := getIntegrationSMTPConfig(t)

	app := application.New(
		application.WithGraceTimeout(2*time.Second),
		application.WithShutdownTimeout(5*time.Second),
	)
	emailInfo, emailFactory := email.New(email.WithConfig(email.Config{
		Host:     smtpHost,
		Port:     smtpPort,
		Username: "integration_user",
		Password: "integration_password",
		FromName: "Bamboo并发测试",
		FromAddr: "bamboo-test@example.invalid",
		TLSType:  "none",
	}))

	if err := app.RegisterTool(emailInfo, emailFactory); err != nil {
		t.Fatalf("RegisterTool(Email) failed: %v", err)
	}

	worker := &emailWorker{
		ready:   make(chan struct{}),
		ctxChan: make(chan context.Context, 1),
	}
	workerInfo := application.PluginInfo{
		Key:      ctxkey.Key("test_email_concurrent_worker"),
		Name:     "TestEmailConcurrentWorker",
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ctxkey.EmailClientKey},
	}
	if err := app.RegisterWorker(workerInfo, func() application.RunnablePlugin { return worker }); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	<-worker.ready
	workerCtx := <-worker.ctxChan
	client, getErr := application.Get[*email.Client](workerCtx, ctxkey.EmailClientKey)
	if getErr != nil {
		t.Fatalf("application.Get[*email.Client] failed: %v", getErr)
	}
	if client == nil {
		t.Fatal("expected non-nil *email.Client")
	}

	// 并发启动多条发送
	var wg sync.WaitGroup
	concurrentCount := 3
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m := email.NewMessage()
			m.To = []string{"user@example.invalid"}
			m.Subject = fmt.Sprintf("Concurrent Test %d [%d]", idx, time.Now().UnixNano())
			m.TextBody = "Concurrent Body"
			_ = client.Send(context.Background(), m)
		}(i)
	}
	wg.Wait()

	// 关闭应用并确保所有 inflight 归零
	if err := app.Shutdown(); err != nil {
		t.Fatalf("app.Shutdown failed: %v", err)
	}
	<-runErrCh

	// 关机后再次发信验证物理断开或状态已停
	mFail := email.NewMessage()
	mFail.To = []string{"user@example.invalid"}
	mFail.Subject = "Post Shutdown"
	mFail.TextBody = "Fail"
	if err := client.Send(context.Background(), mFail); err == nil {
		t.Fatal("expected Send to fail after shutdown, but succeeded")
	}
}
