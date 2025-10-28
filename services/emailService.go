package services

import (
	"fmt"
	"log"
	"os"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client *resend.Client
}

var emailService *EmailService

// InitEmailService initializes the email service with Resend API
func InitEmailService() {
	apiKey := os.Getenv("RESEND_API_KEY")

	if apiKey == "" {
		log.Println("WARNING: RESEND_API_KEY not set. Email service will not be available.")
		return
	}

	emailService = &EmailService{
		client: resend.NewClient(apiKey),
	}

	log.Println("Email service initialized successfully with Resend")
}

// GetEmailService returns the singleton email service instance
func GetEmailService() *EmailService {
	return emailService
}

// SendPasswordResetEmail sends a password reset email with a 6-digit code
func (s *EmailService) SendPasswordResetEmail(toEmail string, code string, firstName string) error {
	if s.client == nil {
		return fmt.Errorf("email service not initialized")
	}

	// Build the email HTML with the 6-digit code
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            text-align: center;
            padding: 20px 0;
            border-bottom: 2px solid #90c590;
        }
        .header h1 {
            color: #90c590;
            margin: 0;
        }
        .content {
            padding: 30px 0;
        }
        .code-container {
            background-color: #f5f5f5;
            border: 2px solid #90c590;
            border-radius: 8px;
            padding: 20px;
            text-align: center;
            margin: 20px 0;
        }
        .code {
            font-size: 32px;
            font-weight: bold;
            letter-spacing: 8px;
            color: #90c590;
            font-family: monospace;
        }
        .warning {
            background-color: #fff3cd;
            border: 1px solid #ffc107;
            border-radius: 4px;
            padding: 15px;
            margin: 20px 0;
        }
        .footer {
            text-align: center;
            padding: 20px 0;
            border-top: 1px solid #ddd;
            font-size: 12px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>prayerloop</h1>
    </div>

    <div class="content">
        <h2>Password Reset Request</h2>

        <p>Hi %s,</p>

        <p>We received a request to reset your prayerloop password. Use the verification code below to complete the password reset process:</p>

        <div class="code-container">
            <div class="code">%s</div>
        </div>

        <p><strong>This code will expire in 15 minutes.</strong></p>

        <div class="warning">
            <p><strong>⚠️ Security Notice:</strong></p>
            <p>If you didn't request a password reset, please ignore this email. Your password will remain unchanged.</p>
        </div>

        <p>Need help? Reply to this email or contact our support team.</p>

        <p>Blessings,<br>The prayerloop Team</p>
    </div>

    <div class="footer">
        <p>&copy; 2025 prayerloop. All rights reserved.</p>
        <p>This is an automated message, please do not reply directly to this email.</p>
    </div>
</body>
</html>
`, firstName, code)

	// Plain text fallback
	textBody := fmt.Sprintf(`
Password Reset Request

Hi %s,

We received a request to reset your prayerloop password. Use the verification code below to complete the password reset process:

Your verification code: %s

This code will expire in 15 minutes.

Security Notice:
If you didn't request a password reset, please ignore this email. Your password will remain unchanged.

Need help? Reply to this email or contact our support team.

Blessings,
The prayerloop Team
`, firstName, code)

	params := &resend.SendEmailRequest{
		From:    os.Getenv("RESEND_FROM_EMAIL"),
		To:      []string{toEmail},
		Subject: "Reset Your prayerloop Password",
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send password reset email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Successfully sent password reset email to %s. Email ID: %s", toEmail, sent.Id)
	return nil
}

// SendWelcomeEmail sends a welcome email to new users (optional - for future use)
func (s *EmailService) SendWelcomeEmail(toEmail string, firstName string) error {
	if s.client == nil {
		return fmt.Errorf("email service not initialized")
	}

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            text-align: center;
            padding: 20px 0;
            border-bottom: 2px solid #90c590;
        }
        .header h1 {
            color: #90c590;
            margin: 0;
        }
        .content {
            padding: 30px 0;
        }
        .footer {
            text-align: center;
            padding: 20px 0;
            border-top: 1px solid #ddd;
            font-size: 12px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Welcome to prayerloop</h1>
    </div>

    <div class="content">
        <h2>Welcome, %s!</h2>

        <p>Thank you for joining prayerloop. We're excited to have you in our community of prayer.</p>

        <p>With prayerloop, you can:</p>
        <ul>
            <li>Create and manage your personal prayer list</li>
            <li>Share prayers with groups and friends</li>
            <li>Track answered prayers and give praise</li>
            <li>Set prayer reminders and build a consistent prayer habit</li>
        </ul>

        <p>Get started by creating your first prayer or joining a prayer group!</p>

        <p>Blessings,<br>The prayerloop Team</p>
    </div>

    <div class="footer">
        <p>&copy; 2025 prayerloop. All rights reserved.</p>
    </div>
</body>
</html>
`, firstName)

	params := &resend.SendEmailRequest{
		From:    os.Getenv("RESEND_FROM_EMAIL"),
		To:      []string{toEmail},
		Subject: "Welcome to prayerloop!",
		Html:    htmlBody,
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send welcome email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Successfully sent welcome email to %s. Email ID: %s", toEmail, sent.Id)
	return nil
}
