package handler

import (
	"context"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	store.Options = &sessions.Options{
		HttpOnly: true,
		MaxAge:   3600,
		Path:     "/",
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	const limit = 5
	page := 1

	session, _ := store.Get(r, "session")
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	adminName := ""
	if name, ok := session.Values["admin_name"].(string); ok {
		adminName = name
	}

	//Getting query parameter
	pageStr := r.URL.Query().Get("page")
	sortField := r.URL.Query().Get("field")
	sortOrder := r.URL.Query().Get("order")

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	switch sortField {
	case "username", "email", "mobile":
	default:
		sortField = "_id"
	}
	switch sortOrder {
	case "asc", "desc":
	default:
		sortOrder = "desc"
	}

	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongo.GetCollection("RegistrationMongo", "New")

	findOptions := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: sortField, Value: getSortOrderValue(sortOrder)}})

	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		render.RenderTemplateWithData(w, "Home.html", EditPageData{
			Error: "Error fetching users from database",
		})
		return
	}
	defer cursor.Close(ctx)

	var users []User
	if err := cursor.All(ctx, &users); err != nil {
		render.RenderTemplateWithData(w, "Home.html", EditPageData{
			Error: "Error decoding users",
		})
		return
	}

	// Count total documents for pagination
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		render.RenderTemplateWithData(w, "Home.html", EditPageData{
			Error: "Error counting users",
		})
		return
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	flash := utils.GetFlashMessage(w, r)

	render.RenderTemplateWithData(w, "Home.html", EditPageData{
		Users:      users,
		Page:       page,
		TotalPages: totalPages,
		Error:      flash,
		Title:      "User Listing",
		SortField:  sortField,
		SortOrder:  sortOrder,
		AdminName:  adminName,
	})
}

func getSortOrderValue(order string) int {
	if order == "asc" {
		return 1
	}
	return -1
}
