package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/sushi-clocks/backend/internal/config"
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendWithAttachments sends an HTML email with optional attachments using standard SMTP.
// If SMTP_HOST is not configured, it logs a simulated dispatch so local/dev environments never crash.
func (s *EmailService) SendWithAttachments(to []string, subject, bodyHTML string, attachments []EmailAttachment) error {
	if len(to) == 0 {
		return nil
	}

	if s.cfg.SMTPHost == "" {
		log.Printf("[EmailService Mock] Dispatching email to %v | Subject: %q | Attachments: %d", to, subject, len(attachments))
		return nil
	}

	from := s.cfg.SMTPFrom
	if from == "" {
		from = "noreply@sushi-clocks.local"
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Top-level MIME headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary())

	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buf.WriteString("\r\n")

	// HTML body part
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Type", "text/html; charset=UTF-8")
	partWriter, err := writer.CreatePart(partHeader)
	if err != nil {
		return fmt.Errorf("create email body part error: %w", err)
	}
	if _, err := partWriter.Write([]byte(bodyHTML)); err != nil {
		return fmt.Errorf("write email body error: %w", err)
	}

	// Attached files
	for _, att := range attachments {
		ct := att.ContentType
		if ct == "" {
			ct = "text/csv"
		}
		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", fmt.Sprintf("%s; name=%q", ct, att.Filename))
		attHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
		attHeader.Set("Content-Transfer-Encoding", "base64")

		attWriter, err := writer.CreatePart(attHeader)
		if err != nil {
			return fmt.Errorf("create attachment part error: %w", err)
		}

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		for len(encoded) > 76 {
			if _, err := attWriter.Write([]byte(encoded[:76] + "\r\n")); err != nil {
				return err
			}
			encoded = encoded[76:]
		}
		if _, err := attWriter.Write([]byte(encoded + "\r\n")); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer error: %w", err)
	}

	var auth smtp.Auth
	if s.cfg.SMTPUser != "" && s.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	if err := smtp.SendMail(addr, auth, from, to, buf.Bytes()); err != nil {
		return fmt.Errorf("smtp sendmail error: %w", err)
	}

	log.Printf("successfully sent archive email to %v (%d attachments)", to, len(attachments))
	return nil
}
