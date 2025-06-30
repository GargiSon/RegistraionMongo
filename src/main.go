package main

import (
	"fmt"
	"go2/mongo"
	"log"
	"net/http"
)

func main() {
	mongo.InitMongoData()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", mongo.LoginHandler)
	http.HandleFunc("/login", mongo.LoginHandler)
	http.HandleFunc("/logout", mongo.LogoutHandler)
	http.HandleFunc("/forgot", mongo.ForgotPasswordHandler)
	http.HandleFunc("/reset", mongo.ResetHandler)

	http.HandleFunc("/register", mongo.RegisterHandler)
	http.HandleFunc("/home", mongo.HomeHandler)
	http.HandleFunc("/edit", mongo.EditHandler)
	http.HandleFunc("/update", mongo.UpdateHandler)
	http.HandleFunc("/delete", mongo.DeleteHandler)

	fmt.Println("Application running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
