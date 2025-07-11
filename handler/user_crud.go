package handler

import (
	"context"
	"encoding/base64"
	"go2/model"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	countries, err := utils.GetCountriesFromDB()
	if err != nil {
		render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
			Error: "Error fetching countries: " + err.Error(),
		})
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		confirm := r.FormValue("confirm")
		email := r.FormValue("email")
		mobile := r.FormValue("mobile")
		address := r.FormValue("address")
		gender := r.FormValue("gender")
		sports := r.Form["sports"]
		dobStr := r.FormValue("dob")
		country := r.FormValue("country")
		joinedSports := strings.Join(sports, ",")

		user := model.User{
			Username: username,
			Email:    email,
			Mobile:   mobile,
			Address:  address,
			Gender:   gender,
			Sports:   joinedSports,
			DOB:      dobStr,
			Country:  country,
		}

		//sports
		sportsMap := make(map[string]bool)
		for _, s := range sports {
			sportsMap[s] = true
		}

		//password
		if password != confirm {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Passwords do not match",
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}

		//dob
		dob, err := time.Parse("2006-01-02", dobStr)
		if err != nil || dob.After(time.Now()) {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Invalid or future DOB",
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}

		//mobile number
		match, err := regexp.MatchString(`^(\+\d{1,3})?\d{10}$`, mobile)
		if err != nil || !match {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Invalid mobile number format",
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}

		//image
		file, _, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			imageBytes, err := io.ReadAll(file)
			if err != nil {
				render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
					Error:     "Error in image uploading",
					Countries: countries,
					User:      user,
					SportsMap: sportsMap,
				})
				return
			}
			user.Image = imageBytes //For storing the image
		}

		//hashing password
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Password hashing failed",
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if mongo.EmailExists(ctx, email) {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Email already used, try a different one.",
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}

		if mongo.MobileExists(ctx, mobile) {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Mobile number already registered",
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}
		user.Password = string(hashed)

		err = mongo.InsertUser(ctx, user)
		if err != nil {
			render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
				Error:     "Registration failed: " + err.Error(),
				Countries: countries,
				User:      user,
				SportsMap: sportsMap,
			})
			return
		}
		utils.SetFlashMessage(w, "User successfully registered!")
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	render.RenderTemplateWithData(w, "Registration.html", model.RegisterPageData{
		Countries: countries,
		Title:     "Add User",
	})
}

func EditHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		render.RenderTemplateWithData(w, "Home.html", model.HomePageData{Error: "Missing user ID"})
		return
	}

	//Convert string ID to ObjectId safely...
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		render.RenderTemplateWithData(w, "Home.html", model.HomePageData{Error: "Invalid ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := mongo.FindUserByID(ctx, objID)
	if err != nil {
		render.RenderTemplateWithData(w, "Home.html", model.HomePageData{
			Error: "User not found",
		})
		return
	}

	if len(user.Image) > 0 {
		user.ImageBase64 = base64.StdEncoding.EncodeToString(user.Image)
	}
	countries, _ := utils.GetCountriesFromDB()

	sportsMap := make(map[string]bool)
	for _, sport := range strings.Split(user.Sports, ",") {
		sport = strings.TrimSpace(sport)
		if sport != "" {
			sportsMap[sport] = true
		}
	}
	if len(user.DOB) > 10 {
		user.DOB = user.DOB[:10]
	}

	render.RenderTemplateWithData(w, "Edit.html", model.EditPageData{
		Title:     "Edit User",
		User:      user,
		Countries: countries,
		SportsMap: sportsMap,
	})
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.SetFlashMessage(w, "Invalid ID")
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	mobile := r.FormValue("mobile")
	address := r.FormValue("address")
	gender := r.FormValue("gender")
	dobStr := r.FormValue("dob")
	country := r.FormValue("country")
	sports := strings.Join(r.Form["sports"], ",")
	removeImage := r.FormValue("remove_image") == "1"

	match, _ := regexp.MatchString(`^(\+\d{1,3})?\d{10}$`, mobile)
	if !match {
		utils.SetFlashMessage(w, "Invalid mobile format")
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	dob, err := time.Parse("2006-01-02", dobStr)
	if err != nil || dob.After(time.Now()) {
		utils.SetFlashMessage(w, "Invalid DOB")
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"username": username,
		"mobile":   mobile,
		"address":  address,
		"gender":   gender,
		"sports":   sports,
		"dob":      dobStr,
		"country":  country,
	}

	file, _, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		imageData, _ := io.ReadAll(file)
		update["image"] = imageData
	} else if removeImage {
		update["image"] = nil
	}

	err = mongo.UpdateUserByID(ctx, objID, update)
	if err != nil {
		utils.SetFlashMessage(w, "Update failed: "+err.Error())
	} else {
		utils.SetFlashMessage(w, "User successfully updated!")
	}
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.SetFlashMessage(w, "Invalid ID")
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = mongo.DeleteUserByID(ctx, objID)
	if err != nil {
		utils.SetFlashMessage(w, "Error deleting user")
	} else {
		utils.SetFlashMessage(w, "User deleted!")
	}
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}
