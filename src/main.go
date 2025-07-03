package main

import (
	"fmt"
	"go2/handler"
	"go2/mongo"
	"log"
	"net/http"
)

func main() {
	// Run this single time
	// mongo.Connect()

	handler.InitSession()
	mongo.InitMongoData()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", handler.LoginHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/logout", handler.LogoutHandler)
	http.HandleFunc("/forgot", handler.ForgotPasswordHandler)
	http.HandleFunc("/reset", handler.ResetHandler)

	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/home", handler.HomeHandler)
	http.HandleFunc("/edit", handler.EditHandler)
	http.HandleFunc("/update", handler.UpdateHandler)
	http.HandleFunc("/delete", handler.DeleteHandler)

	fmt.Println("Application running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
