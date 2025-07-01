package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
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
		render.RenderTemplateWithData(w, "Login.html", EditPageData{})
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongo.GetCollection("RegistrationMongo", "admins")

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
		render.RenderTemplateWithData(w, "Forgot.html", EditPageData{Info: utils.GetFlashMessage(w, r)})
		return
	}

	email := r.FormValue("email")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongo.GetCollection("RegistrationMongo", "admins")
	count, err := collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil || count == 0 {
		utils.SetFlashMessage(w, "Email not Found")
		http.Redirect(w, r, "/forgot", http.StatusSeeOther)
		return
	}

	ts := fmt.Sprint(time.Now().Unix())
	hash := sha256.Sum256([]byte(email + ts + resetsecret))
	token := hex.EncodeToString(hash[:])

	link := fmt.Sprintf("http://localhost:8080/reset?email=%s&ts=%s&token=%s", url.QueryEscape(email), ts, token)
	if err := sendResetEmail(email, link); err != nil {
		log.Println("Failed to send email:", err)
		utils.SetFlashMessage(w, "Failed to send reset link. Try again.")
	} else {
		utils.SetFlashMessage(w, "Reset link sent! Check your email.")
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

		collection := mongo.GetCollection("RegistrationMongo", "admins")
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
		utils.SetFlashMessage(w, "Password updated successfully.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	render.RenderTemplateWithData(w, "Reset.html", EditPageData{
		Email: email, Ts: ts, Token: token,
	})
}
