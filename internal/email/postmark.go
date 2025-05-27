package email

import (
	"fmt"

	"github.com/keighl/postmark"
)

// PostmarkConfig contains configuration for the Postmark service
type PostmarkConfig struct {
	ServerToken string `envconfig:"POSTMARK_SERVER_TOKEN" required:"true"`
	FromEmail   string `envconfig:"POSTMARK_EMAIL_FROM" default:"noreply@tofudns.net"`
}

// PostmarkService implements the email service using Postmark
type PostmarkService struct {
	client    *postmark.Client
	fromEmail string
}

// NewPostmarkService creates a new Postmark email service
func NewPostmarkService(config PostmarkConfig) *PostmarkService {
	client := postmark.NewClient(config.ServerToken, "")
	return &PostmarkService{
		client:    client,
		fromEmail: config.FromEmail,
	}
}

// SendOTP sends an OTP code to the specified email
func (s *PostmarkService) SendOTP(email, otp string) error {
	emailReq := postmark.Email{
		From:       s.fromEmail,
		To:         email,
		Subject:    "Your TofuDNS Verification Code",
		TextBody:   fmt.Sprintf("Your verification code is: %s\nThis code will expire in 10 minutes.", otp),
		HtmlBody:   fmt.Sprintf("<h2>Your TofuDNS Verification Code</h2><p>Your verification code is: <strong>%s</strong></p><p>This code will expire in 10 minutes.</p>", otp),
		TrackOpens: true,
	}

	_, err := s.client.SendEmail(emailReq)
	return err
}
