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
	http.HandleFunc("/forgot", handler.ForgotPasswordHandler)
	http.HandleFunc("/reset", handler.ResetHandler)
	http.HandleFunc("/logout", handler.LogoutHandler)

	// Protected routes
	http.HandleFunc("/home", handler.RequireLogin(handler.HomeHandler))
	http.HandleFunc("/edit", handler.RequireLogin(handler.EditHandler))
	http.HandleFunc("/register", handler.RequireLogin(handler.RegisterHandler))
	http.HandleFunc("/update", handler.RequireLogin(handler.UpdateHandler))
	http.HandleFunc("/delete", handler.RequireLogin(handler.DeleteHandler))

	fmt.Println("Application running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
