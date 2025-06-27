package mongo

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

// this is used for html queries not for mongodb so, bson is not required!
type EditPageData struct {
	User       User
	Countries  []string
	SportsMap  map[string]bool
	Error      string
	Title      string
	Users      []User
	Page       int
	TotalPages int
	Info       string
	Email      string
	Ts         string
	Token      string
	SortField  string
	SortOrder  string
	AdminName  string
}
