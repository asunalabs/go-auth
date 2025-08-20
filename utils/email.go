package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailSender defines the behaviour for sending emails. Implementations should
// be safe for concurrent use by multiple goroutines.
type EmailSender interface {
	Send(ctx context.Context, to []string, subject, body string) error
}

// SMTPClient sends emails using an SMTP server. It reads configuration from
// environment variables: SMTP_HOST, SMTP_PORT, SMTP_EMAIL, SMTP_PASSWORD.
// If not provided, sensible defaults are used (smtp.gmail.com:587).
type SMTPClient struct {
	host     string
	port     string
	email    string
	password string
	auth     smtp.Auth
	addr     string
	timeout  time.Duration
	useTLS   bool // whether to use implicit TLS (port 465)
}

// NewSMTPClient builds an SMTPClient from environment variables.
func NewSMTPClient() *SMTPClient {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = "smtp.gmail.com"
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	email := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	auth := smtp.PlainAuth("", email, password, host)
	useTLS := port == "465"

	return &SMTPClient{
		host:     host,
		port:     port,
		email:    email,
		password: password,
		auth:     auth,
		addr:     net.JoinHostPort(host, port),
		timeout:  10 * time.Second,
		useTLS:   useTLS,
	}
}

// Send composes and sends a plain-text email to one or more recipients.
// It validates inputs and returns detailed errors for hard failures.
func (s *SMTPClient) Send(ctx context.Context, to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients provided")
	}
	if s.email == "" {
		return fmt.Errorf("sender email (SMTP_EMAIL) is not configured")
	}

	header := make(map[string]string)
	header["From"] = s.email
	header["To"] = strings.Join(to, ", ")
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// Dial the SMTP server with a timeout
	var conn net.Conn
	var err error
	d := net.Dialer{Timeout: s.timeout}

	if s.useTLS {
		// implicit TLS (port 465)
		tlsConfig := &tls.Config{ServerName: s.host}
		conn, err = tls.DialWithDialer(&d, "tcp", s.addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial: %w", err)
		}
	} else {
		conn, err = d.DialContext(ctx, "tcp", s.addr)
		if err != nil {
			return fmt.Errorf("dial smtp: %w", err)
		}
	}

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}

	// If using STARTTLS (typically port 587), upgrade the connection.
	if !s.useTLS {
		tlsConfig := &tls.Config{ServerName: s.host}
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err = c.StartTLS(tlsConfig); err != nil {
				c.Close()
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}

	// Authenticate if credentials are present
	if s.email != "" && s.password != "" {
		if err = c.Auth(s.auth); err != nil {
			c.Close()
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err = c.Mail(s.email); err != nil {
		c.Close()
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err = c.Rcpt(rcpt); err != nil {
			c.Close()
			return fmt.Errorf("rcpt %s: %w", rcpt, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		c.Close()
		return fmt.Errorf("data: %w", err)
	}
	if _, err = w.Write([]byte(msg.String())); err != nil {
		w.Close()
		c.Close()
		return fmt.Errorf("write message: %w", err)
	}
	if err = w.Close(); err != nil {
		c.Close()
		return fmt.Errorf("close write: %w", err)
	}

	if err = c.Quit(); err != nil {
		return fmt.Errorf("quit: %w", err)
	}
	return nil
}
