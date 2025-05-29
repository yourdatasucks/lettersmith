package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"

	"github.com/yourdatasucks/lettersmith/internal/config"
)

type Client struct {
	config *config.EmailConfig
}

func NewClient(cfg *config.EmailConfig) *Client {
	return &Client{
		config: cfg,
	}
}

func (c *Client) SendEmail(to, subject, body string) error {
	switch c.config.Provider {
	case "smtp":
		return c.sendSMTP(to, subject, body)
	case "sendgrid":
		return c.sendSendGrid(to, subject, body)
	case "mailgun":
		return c.sendMailgun(to, subject, body)
	default:
		return fmt.Errorf("unsupported email provider: %s", c.config.Provider)
	}
}

func (c *Client) sendSMTP(to, subject, body string) error {

	if c.config.SMTP.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if c.config.SMTP.Port == 0 {
		return fmt.Errorf("SMTP port is required")
	}
	if c.config.SMTP.Username == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if c.config.SMTP.Password == "" {
		return fmt.Errorf("SMTP password is required")
	}

	from := c.config.SMTP.From
	if from == "" {
		from = c.config.SMTP.Username
	}

	message := fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", to, subject, body)

	addr := fmt.Sprintf("%s:%d", c.config.SMTP.Host, c.config.SMTP.Port)

	log.Printf("Connecting to SMTP server: %s", addr)

	auth := smtp.PlainAuth("", c.config.SMTP.Username, c.config.SMTP.Password, c.config.SMTP.Host)

	if c.config.SMTP.Host == "127.0.0.1" || c.config.SMTP.Host == "localhost" {

		return c.sendSMTPWithTLS(addr, auth, from, []string{to}, []byte(message), true)
	}

	return c.sendSMTPWithTLS(addr, auth, from, []string{to}, []byte(message), false)
}

func (c *Client) sendSMTPWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, allowInsecure bool) error {

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         c.config.SMTP.Host,
			InsecureSkipVerify: allowInsecure, // Allow for local bridges with self-signed certs
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data connection: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data connection: %w", err)
	}

	return client.Quit()
}

func (c *Client) sendSendGrid(to, subject, body string) error {

	return fmt.Errorf("SendGrid integration not yet implemented")
}

func (c *Client) sendMailgun(to, subject, body string) error {

	return fmt.Errorf("mailgun integration not yet implemented")
}

func (c *Client) TestConnection() error {
	switch c.config.Provider {
	case "smtp":
		return c.testSMTPConnection()
	case "sendgrid":
		return c.testSendGridConnection()
	case "mailgun":
		return c.testMailgunConnection()
	default:
		return fmt.Errorf("unsupported email provider: %s", c.config.Provider)
	}
}

func (c *Client) testSMTPConnection() error {
	addr := fmt.Sprintf("%s:%d", c.config.SMTP.Host, c.config.SMTP.Port)

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server %s: %w", addr, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         c.config.SMTP.Host,
			InsecureSkipVerify: c.config.SMTP.Host == "127.0.0.1" || c.config.SMTP.Host == "localhost",
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	auth := smtp.PlainAuth("", c.config.SMTP.Username, c.config.SMTP.Password, c.config.SMTP.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return client.Quit()
}

func (c *Client) testSendGridConnection() error {

	return fmt.Errorf("sendGrid testing not yet implemented")
}

func (c *Client) testMailgunConnection() error {

	return fmt.Errorf("mailgun testing not yet implemented")
}

func (c *Client) GetConnectionInfo() string {
	switch c.config.Provider {
	case "smtp":
		return fmt.Sprintf("SMTP: %s:%d (user: %s)",
			c.config.SMTP.Host, c.config.SMTP.Port, c.config.SMTP.Username)
	case "sendgrid":
		return fmt.Sprintf("SendGrid API (from: %s)", c.config.SendGrid.From)
	case "mailgun":
		return fmt.Sprintf("Mailgun API (domain: %s)", c.config.Mailgun.Domain)
	default:
		return "Unknown provider"
	}
}
