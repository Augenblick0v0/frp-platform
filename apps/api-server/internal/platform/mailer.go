package platform

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

type Mailer interface {
	Send(to, subject, textBody string) error
	SendVerificationCode(to, code, purpose string) error
}

type LogMailer struct{}

func (LogMailer) Send(to, subject, textBody string) error {
	log.Printf("mail dry-run to=%s subject=%q body=%q", to, subject, textBody)
	return nil
}
func (m LogMailer) SendVerificationCode(to, code, purpose string) error {
	return m.Send(to, "FRP 平台邮箱验证码", VerificationMailBody(code, purpose))
}

type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	FromName   string
	UseTLS     bool
	SkipVerify bool
}

type SMTPMailer struct{ cfg SMTPConfig }

func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer { return &SMTPMailer{cfg: cfg} }

func MailerFromEnv() Mailer {
	cfg := SMTPConfig{
		Host:       os.Getenv("SMTP_HOST"),
		Username:   os.Getenv("SMTP_USERNAME"),
		Password:   os.Getenv("SMTP_PASSWORD"),
		FromEmail:  os.Getenv("SMTP_FROM_EMAIL"),
		FromName:   getenv("SMTP_FROM_NAME", "FRP Tunnel Platform"),
		UseTLS:     getenv("SMTP_TLS", "true") != "false",
		SkipVerify: os.Getenv("SMTP_SKIP_VERIFY") == "true",
	}
	cfg.Port, _ = strconv.Atoi(getenv("SMTP_PORT", "587"))
	if cfg.Host == "" || cfg.FromEmail == "" {
		return LogMailer{}
	}
	return NewSMTPMailer(cfg)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (m *SMTPMailer) SendVerificationCode(to, code, purpose string) error {
	return m.Send(to, "FRP 平台邮箱验证码", VerificationMailBody(code, purpose))
}

func VerificationMailBody(code, purpose string) string {
	label := map[string]string{"register": "注册", "reset_password": "重置密码", "login": "登录"}[purpose]
	if label == "" {
		label = "操作"
	}
	return fmt.Sprintf("您的 FRP 平台%s验证码是：%s\n\n验证码 10 分钟内有效。如非本人操作，请忽略本邮件。\n", label, code)
}

func (m *SMTPMailer) Send(to, subject, textBody string) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient required")
	}
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	fromHeader := m.cfg.FromEmail
	if m.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", encodeHeader(m.cfg.FromName), m.cfg.FromEmail)
	}
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", encodeHeader(subject))
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&msg, "\r\n%s", textBody)
	auth := smtp.Auth(nil)
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	if m.cfg.UseTLS && m.cfg.Port == 465 {
		return m.sendImplicitTLS(addr, auth, m.cfg.FromEmail, []string{to}, msg.Bytes())
	}
	return smtp.SendMail(addr, auth, m.cfg.FromEmail, []string{to}, msg.Bytes())
}

func (m *SMTPMailer) sendImplicitTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host, InsecureSkipVerify: m.cfg.SkipVerify})
	if err != nil {
		return err
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return err
	}
	defer c.Quit()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func encodeHeader(s string) string { return "=?UTF-8?B?" + base64Encode([]byte(s)) + "?=" }

const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func base64Encode(src []byte) string {
	var out strings.Builder
	for i := 0; i < len(src); i += 3 {
		var n uint32
		remain := len(src) - i
		n |= uint32(src[i]) << 16
		if remain > 1 {
			n |= uint32(src[i+1]) << 8
		}
		if remain > 2 {
			n |= uint32(src[i+2])
		}
		out.WriteByte(b64[(n>>18)&63])
		out.WriteByte(b64[(n>>12)&63])
		if remain > 1 {
			out.WriteByte(b64[(n>>6)&63])
		} else {
			out.WriteByte('=')
		}
		if remain > 2 {
			out.WriteByte(b64[n&63])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}
