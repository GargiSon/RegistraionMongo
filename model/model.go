package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Username    string             `bson:"username"`
	Email       string             `bson:"email"`
	Password    string             `bson:"password"`
	Mobile      string             `bson:"mobile"`
	Address     string             `bson:"address"`
	Gender      string             `bson:"gender"`
	Sports      string             `bson:"sports"`
	DOB         string             `bson:"dob"`
	Country     string             `bson:"country"`
	Image       []byte             `bson:"image,omitempty"`
	ImageBase64 string
}

type Admin struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Email    string             `bson:"email"`
	Password string             `bson:"password"`
}

type PasswordResetToken struct {
	UserID      primitive.ObjectID `bson:"user_id"`
	TokenHash   string             `bson:"token"`
	TokenExpiry int64              `bson:"token_expiry"`
}

// this is used for html queries not for mongodb so, bson is not required!
type RegisterPageData struct {
	User      User
	Countries []string
	SportsMap map[string]bool
	Error     string
	Title     string
}

type HomePageData struct {
	Users      []User
	Page       int
	TotalPages int
	Error      string
	Title      string
	SortField  string
	SortOrder  string
	AdminName  string
}

type EditPageData struct {
	Title     string
	User      User
	Countries []string
	SportsMap map[string]bool
	Error     string
}

type EmailData struct {
	ResetLink string
}

type LoginPageData struct {
	Error string
	Title string
}

type ForgotPageData struct {
	Info  string
	Title string
	Error string
}

type ResetPageData struct {
	Error string
	Token string
	Title string
	Info  string
}
