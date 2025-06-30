package mongo

type Admin struct {
	Email    string `bson:"email"`
	Password string `bson:"password"`
}
