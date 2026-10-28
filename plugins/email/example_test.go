package email_test

import (
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/plugins/email/v2"
)

// ExampleNew 展示如何通过显式配置构造 Email 插件定义。
func ExampleNew() {
	info, _ := email.New(
		email.WithConfig(email.Config{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "notifier",
			Password: "secret_password",
			FromAddr: "service@example.com",
			FromName: "Bamboo Service",
		}),
	)

	fmt.Println(info.Key)
	fmt.Println(info.Mode == application.ModeTool)
	fmt.Println(info.Provides[0] == ctxkey.EmailClientKey)
	// Output:
	// plugin_email
	// true
	// true
}

// ExampleNewMessage 展示如何构造包含主题、正文与内存附件的邮件消息。
func ExampleNewMessage() {
	msg := email.NewMessage()
	msg.To = []string{"user@example.com"}
	msg.Subject = "系统日报"
	msg.TextBody = "请查收今日系统日报。"
	msg.AttachReader("report.txt", strings.NewReader("系统运行正常"))

	fmt.Println(msg.Subject)
	fmt.Println(len(msg.To))
	fmt.Println(len(msg.Attachments))
	// Output:
	// 系统日报
	// 1
	// 1
}
