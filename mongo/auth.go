package mongo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go2/render"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("super-secret-session-key"))

const resetsecret = "hubjinkom"

func sendResetEmail(toEmail, resetLink string) error {
	email := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	auth := smtp.PlainAuth("", email, password, "smtp.gmail.com")

	subject := "Subject: Password Reset Link\n"
	headers := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 40px 0;">
		<div style="max-width: 600px; margin: auto; background-color: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
			<p style="font-size: 18px;">Hello,</p>
			<p style="font-size: 16px;">Click the button below to reset your password:</p>
			<p style="text-align: center;">
				<a href="%s" style="display: inline-block; background-color: #007BFF; color: white; padding: 12px 20px; text-decoration: none; border-radius: 5px; font-size: 16px;">Reset Password</a>
			</p>
			<p style="font-size: 14px;">Or copy and paste this URL into your browser:</p>
			<p style="word-break: break-all; font-size: 14px; color: #333;">%s</p>
			<br>
			<p style="font-size: 14px;">If you didn’t request this, please ignore this email.</p>
			<p style="font-size: 14px;">Thanks,<br><strong>Your Team</strong></p>
		</div>
	</body>
	</html>`, resetLink, resetLink)

	msg := []byte(subject + headers + body)

	return smtp.SendMail("smtp.gmail.com:587", auth, email, []string{toEmail}, msg)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render.RenderTemplateWithData(w, "Login.html", EditPageData{})
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("RegistrationMongo", "AdminNew")

	var result struct {
		Password string `bson:"password"`
	}
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&result)
	if err != nil {
		render.RenderTemplateWithData(w, "Login.html", EditPageData{
			Error: "Invalid email or password",
		})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(result.Password), []byte(password)) != nil {
		render.RenderTemplateWithData(w, "Login.html", EditPageData{
			Error: "Invalid email or password",
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
		render.RenderTemplateWithData(w, "Login.html", EditPageData{
			Error: "Failed to start session",
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
		render.RenderTemplateWithData(w, "Forgot.html", EditPageData{Info: getFlashMessage(w, r)})
		return
	}

	email := r.FormValue("email")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("RegistrationMongo", "AdminNew")
	count, err := collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil || count == 0 {
		setFlashMessage(w, "Email not Found")
		http.Redirect(w, r, "/forgot", http.StatusSeeOther)
		return
	}

	ts := fmt.Sprint(time.Now().Unix())
	hash := sha256.Sum256([]byte(email + ts + resetsecret))
	token := hex.EncodeToString(hash[:])

	link := fmt.Sprintf("http://localhost:8080/reset?email=%s&ts=%s&token=%s", url.QueryEscape(email), ts, token)
	if err := sendResetEmail(email, link); err != nil {
		log.Println("Failed to send email:", err)
		setFlashMessage(w, "Failed to send reset link. Try again.")
	} else {
		setFlashMessage(w, "Reset link sent! Check your email.")
	}
	http.Redirect(w, r, "/forgot", http.StatusSeeOther)
}

func ResetHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	ts := r.FormValue("ts")
	token := r.FormValue("token")

	expectedHash := sha256.Sum256([]byte(email + ts + resetsecret))
	expectedToken := hex.EncodeToString(expectedHash[:])

	if token != expectedToken {
		http.Error(w, "Invalid or tampered reset link.", http.StatusUnauthorized)
		return
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Now().Unix()-tsInt > 900 {
		http.Error(w, "Reset link has expired.", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		newPass := r.FormValue("password")
		confirm := r.FormValue("confirm")
		if newPass != confirm {
			render.RenderTemplateWithData(w, "Reset.html", EditPageData{
				Error: "Passwords do not match.",
				Email: email, Ts: ts, Token: token,
			})
			return
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		collection := GetCollection("RegistrationMongo", "AdminNew")
		_, err := collection.UpdateOne(ctx, bson.M{"email": email}, bson.M{
			"$set": bson.M{"password": string(hashed)},
		})

		if err != nil {
			render.RenderTemplateWithData(w, "Reset.html", EditPageData{
				Error: "Failed to update password",
				Email: email, Ts: ts, Token: token,
			})
			return
		}
		setFlashMessage(w, "Password updated successfully.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	render.RenderTemplateWithData(w, "Reset.html", EditPageData{
		Email: email, Ts: ts, Token: token,
	})
}
