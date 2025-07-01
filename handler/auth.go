package handler

import (
	"bytes"
	"context"
	"fmt"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("super-secret-session-key"))

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

	//1. Admin Enters email through form
	email := r.FormValue("email")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//2. Find email in database and Display Generic message after getting email to provide authentication
	adminColl := mongo.GetCollection("RegistrationMongo", "admins")
	var admin struct {
		ID    primitive.ObjectID `bson:"_id"`
		Email string             `bson:"email"`
	}
	err := adminColl.FindOne(ctx, bson.M{"email": email}).Decode(&admin)
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
	tokenColl := mongo.GetCollection("RegistrationMongo", "password_reset_tokens")
	_, _ = tokenColl.InsertOne(ctx, bson.M{
		"user_id":      admin.ID,
		"token":        tokenHash,
		"token_expiry": expiry,
	})

	//5. Sending email
	link := fmt.Sprintf("http://localhost:8080/reset?token=%s", rawToken)
	_ = sendResetEmail(email, link)
	http.Redirect(w, r, "/forgot", http.StatusSeeOther)
}

func ResetHandler(w http.ResponseWriter, r *http.Request) {
	//1. Get rawToken
	rawToken := r.URL.Query().Get("token")

	tokenHash := utils.HashSHA256(rawToken)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tokenColl := mongo.GetCollection("RegistrationMongo", "password_reset_tokens")

	var tokenData PasswordResetToken
	err := tokenColl.FindOne(ctx, bson.M{"token": tokenHash}).Decode(&tokenData)
	if err != nil {
		render.RenderTemplateWithData(w, "Reset.html", EditPageData{
			Error: "Invalid or expired token",
		})
		return
	}

	if time.Now().Unix() > tokenData.TokenExpiry {
		// Token expired, clean up
		tokenColl.DeleteMany(ctx, bson.M{"user_id": tokenData.UserID})
		render.RenderTemplateWithData(w, "Reset.html", EditPageData{
			Error: "Token Expired",
		})
		return
	}

	if r.Method == http.MethodPost {
		newPass := r.FormValue("password")
		confirm := r.FormValue("confirm")
		if newPass != confirm {
			render.RenderTemplateWithData(w, "Reset.html", EditPageData{
				Error: "Passwords do not match.",
				Token: rawToken,
			})
			return
		}

		hashedPass, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)

		adminColl := mongo.GetCollection("RegistrationMongo", "admins")
		_, err = adminColl.UpdateByID(ctx, tokenData.UserID, bson.M{
			"$set": bson.M{"password": string(hashedPass)},
		})
		if err != nil {
			render.RenderTemplateWithData(w, "Reset.html", EditPageData{
				Error: "Failed to update password.",
				Token: rawToken,
			})
			return
		}
		tokenColl.DeleteMany(ctx, bson.M{"user_id": tokenData.UserID})
		utils.SetFlashMessage(w, "Password updated successfully. Please log in.")
		http.Redirect(w, r, "/", http.StatusSeeOther) // Redirect to login page
		return
	}
	render.RenderTemplateWithData(w, "Reset.html", EditPageData{
		Token: rawToken,
	})
}
