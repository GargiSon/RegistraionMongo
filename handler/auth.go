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
	"net/smtp"
	"os"
	"strings"
	"text/template"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func sendResetEmail(toEmail, resetLink string) error {
	email := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	auth := smtp.PlainAuth("", email, password, "smtp.gmail.com")

	subject := "Subject: Password Reset Link\n"
	headers := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	tmpl, err := template.ParseFiles("templates/reset_email.html")
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	var bodyBuffer bytes.Buffer
	err = tmpl.Execute(&bodyBuffer, struct{ Link string }{Link: resetLink})
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	msg := []byte(subject + headers + bodyBuffer.String())

	return smtp.SendMail("smtp.gmail.com:587", auth, email, []string{toEmail}, msg)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render.RenderTemplateWithData(w, "Login.html", model.LoginPageData{
			Title: "Login",
		})
		return
	}

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

	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password))
	if err != nil {
		render.RenderTemplateWithData(w, "Login.html", model.LoginPageData{
			Error: "Invalid PASSWORD",
			Title: "Login",
		})
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = true
	session.Values["email"] = email
	parts := strings.Split(email, "@")
	session.Values["admin_name"] = parts[0]
	err = session.Save(r, w)
	if err != nil {
		render.RenderTemplateWithData(w, "Login.html", model.LoginPageData{
			Error: "Failed to start session",
			Title: "Login",
		})
		return
	}

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render.RenderTemplateWithData(w, "Forgot.html", model.ForgotPageData{
			Info:  utils.GetFlashMessage(w, r),
			Title: "Forgot Password",
			Error: "",
		})
		return
	}

	//1. Admin Enters email through form
	email := r.FormValue("email")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//2. Find email in database and Display Generic message after getting email to provide authentication
	admin, err := mongo.GetAdminByEmail(ctx, email)
	utils.SetFlashMessage(w, "If the email exists, a reset link will be sent.")
	//If the admin donot exist, do not proceed further, silently redirect
	if err != nil {
		http.Redirect(w, r, "/forgot", http.StatusSeeOther)
		return
	}

	//3. Generate secure token(generate + hash + expiry)
	rawToken := utils.GenerateSecureToken(64)
	tokenHash := utils.HashSHA256(rawToken)
	expiry := time.Now().Add(15 * time.Minute).Unix()

	//4. Store in password reset tokens collection
	_ = mongo.InsertResetToken(ctx, admin.ID, tokenHash, expiry)

	// 5. Get the base link from environment
	baseURL := os.Getenv("AUTH_LINK")
	if baseURL == "" {
		baseURL = "http://localhost:8080/reset?token="
	}

	link := fmt.Sprintf("%s%s", baseURL, rawToken)

	// 6. Send the reset email
	fmt.Println("About to send email to:", email) //To verify that it is reacble or not
	err = sendResetEmail(email, link)
	if err != nil {
		fmt.Println("Failed to send email:", err)
	} else {
		fmt.Println("Email sent to:", email)
	}

	http.Redirect(w, r, "/forgot", http.StatusSeeOther)
}

func ResetHandler(w http.ResponseWriter, r *http.Request) {
	// Get rawToken
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
		_ = mongo.DeleteResetTokensByUserID(ctx, tokenData.UserID) //Expired token clean up
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

func TempLoginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = true
	session.Values["email"] = "admin@temp.com"
	session.Values["admin_name"] = "tempadmin"
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to start session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}
