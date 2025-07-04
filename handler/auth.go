package handler

import (
	"bytes"
	"context"
	"fmt"
	"go2/model"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"net/http"
	"os"
	"text/template"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
)

func sendResetEmail(toEmail, resetLink string) error {
	tmpl, err := template.ParseFiles("templates/reset_email.html")
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	var bodyBuffer bytes.Buffer
	err = tmpl.Execute(&bodyBuffer, struct{ Link string }{Link: resetLink})
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	email := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	m := gomail.NewMessage()
	m.SetHeader("From", email)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Password Reset Link")
	m.SetBody("text/html", bodyBuffer.String())

	//Configures SMTP dialer using Gmail
	d := gomail.NewDialer("smtp.gmail.com", 587, email, password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send mail: %w", err)
	}
	return nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	setNoCacheHeaders(w)

	if r.Method == http.MethodGet {
		// If already logged in, redirect to home
		if _, ok := GetSessionEmail(r); ok {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
		render.RenderTemplateWithData(w, "Login.html", model.LoginPageData{
			Title: "Login",
		})
		return
	}

	//POST Logic
	email := r.FormValue("email")
	password := r.FormValue("password")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := mongo.GetAdminByEmail(ctx, email)

	if err != nil || bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)) != nil {
		render.RenderTemplateWithData(w, "Login.html", model.LoginPageData{
			Error: "Invalid email or password",
			Title: "Login",
		})
		return
	}

	// Set session using in-memory map and cookie, Login successful redirect to home
	SetSession(w, email)
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	setNoCacheHeaders(w)
	ClearSession(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Get request, displays forget password form
	if r.Method == http.MethodGet {
		render.RenderTemplateWithData(w, "Forgot.html", model.ForgotPageData{
			Info:  utils.GetFlashMessage(w, r),
			Title: "Forgot Password",
			Error: "",
		})
		return
	}

	// Admin enters email through form
	email := r.FormValue("email")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := mongo.GetAdminByEmail(ctx, email)
	utils.SetFlashMessage(w, "If the email exists, a reset link will be sent.")
	if err != nil {
		fmt.Println("Email not found in DB:", email)
		http.Redirect(w, r, "/forgot", http.StatusSeeOther)
		return
	}

	// Generate secure token (generate + hash + expiry)
	rawToken := utils.GenerateSecureToken(64)
	tokenHash := utils.HashSHA256(rawToken)
	expiry := time.Now().Add(15 * time.Minute).Unix()

	// Store in password reset tokens collection
	err = mongo.InsertResetToken(ctx, admin.ID, tokenHash, expiry)
	if err != nil {
		fmt.Println("Failed to store token:", err)
		return
	}
	fmt.Println("Token stored in DB")

	// Get the base link from environment
	baseURL := os.Getenv("AUTH_LINK")
	if baseURL == "" {
		baseURL = "http://localhost:8080/reset?token="
	}
	link := baseURL + rawToken

	// Send reset email
	err = sendResetEmail(email, link)
	if err != nil {
		fmt.Println("Failed to send email:", err)
	} else {
		fmt.Println("Email sent to:", email)
	}

	http.Redirect(w, r, "/forgot", http.StatusSeeOther)
}

func ResetHandler(w http.ResponseWriter, r *http.Request) {
	setNoCacheHeaders(w)

	rawToken := r.URL.Query().Get("token")
	tokenHash := utils.HashSHA256(rawToken)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tokenData, err := mongo.FindResetToken(ctx, tokenHash)
	if err != nil {
		render.RenderTemplateWithData(w, "Reset.html", model.ResetPageData{
			Error: "Invalid or expired token",
			Title: "Reset Password",
		})
		return
	}

	if time.Now().Unix() > tokenData.TokenExpiry {
		_ = mongo.DeleteResetTokensByUserID(ctx, tokenData.UserID)
		render.RenderTemplateWithData(w, "Reset.html", model.ResetPageData{
			Error: "Token Expired",
			Title: "Reset Password",
		})
		return
	}

	if r.Method == http.MethodPost {
		newPass := r.FormValue("password")
		confirm := r.FormValue("confirm")
		if newPass != confirm {
			render.RenderTemplateWithData(w, "Reset.html", model.ResetPageData{
				Error: "Passwords do not match.",
				Token: rawToken,
				Title: "Reset Password",
			})
			return
		}

		hashedPass, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
		err := mongo.UpdateAdminPassword(ctx, tokenData.UserID, string(hashedPass))
		if err != nil {
			render.RenderTemplateWithData(w, "Reset.html", model.ResetPageData{
				Error: "Failed to update password.",
				Token: rawToken,
				Title: "Reset Password",
			})
			return
		}

		_ = mongo.DeleteResetTokensByUserID(ctx, tokenData.UserID)
		utils.SetFlashMessage(w, "Password updated successfully.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	render.RenderTemplateWithData(w, "Reset.html", model.ResetPageData{
		Token: rawToken,
		Title: "Reset Password",
	})
}
