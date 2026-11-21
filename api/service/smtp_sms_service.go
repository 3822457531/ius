package service

import (
	"bytes"
	"chatplus/core/types"
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
)

type SmtpService struct {
	config *types.SmtpConfig
}

func NewSmtpService(appConfig *types.AppConfig) *SmtpService {
	return &SmtpService{
		config: &appConfig.SmtpConfig,
	}
}

//
//func (s *SmtpService) SendVerifyCode(to string, code int) error {
//	subject := "GPTAI注册验证码"
//	body := fmt.Sprintf("您正在注册 GPT AI 账户，注册验证码为 %d，请不要告诉他人。如非本人操作，请忽略此邮件。", code)
//
//	// 设置SMTP客户端配置
//	auth := smtp.PlainAuth("", s.config.From, s.config.Password, s.config.Host)
//
//	// 对主题进行MIME编码
//	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
//	// 组装邮件
//	message := bytes.NewBuffer(nil)
//	message.WriteString(fmt.Sprintf("From: \"%s\" <%s>\r\n", s.config.AppName, s.config.From))
//	message.WriteString(fmt.Sprintf("To: %s\r\n", to))
//	message.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
//	message.WriteString("\r\n" + body)
//
//	// 发送邮件
//	// 发送邮件
//	err := smtp.SendMail(s.config.Host+":"+fmt.Sprint(s.config.Port), auth, s.config.From, []string{to}, message.Bytes())
//	if err != nil {
//		return fmt.Errorf("error sending email: %v", err)
//	}
//	return nil
//}

func (s *SmtpService) SendVerifyCode(to string, code int) error {
	subject := "GPTAI注册验证码"
	body := fmt.Sprintf("您正在注册 GPT AI 账户，注册验证码为 %d，请不要告诉他人。如非本人操作，请忽略此邮件。", code)

	// 设置SMTP客户端配置
	auth := smtp.PlainAuth("", s.config.From, s.config.Password, s.config.Host)

	// 对主题进行MIME编码
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)

	// 组装邮件
	message := bytes.NewBuffer(nil)
	message.WriteString(fmt.Sprintf("From: \"%s\" <%s>\r\n", s.config.AppName, s.config.From))
	message.WriteString(fmt.Sprintf("To: %s\r\n", to))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	message.WriteString("\r\n" + body)

	// 设置TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // 如果你有合适的证书，可以将其设置为 false
		ServerName:         s.config.Host,
	}

	// 建立TLS连接
	conn, err := tls.Dial("tcp", s.config.Host+":465", tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to establish tls connection: %v", err)
	}

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %v", err)
	}

	// 使用auth认证
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp authentication failed: %v", err)
	}

	// 发送邮件
	if err = client.Mail(s.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %v", err)
	}

	// 获取写入数据
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get writer: %v", err)
	}

	_, err = writer.Write(message.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	client.Quit()

	return nil
}
