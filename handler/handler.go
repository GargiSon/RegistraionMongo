package handler

import (
	"context"
	"go2/model"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"net/http"
	"strconv"
	"time"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	users, total, err := mongo.GetPaginatedUsers(ctx, page, userPageLimit, sortField, sortOrder)
	if err != nil {
		render.RenderTemplateWithData(w, "Home.html", model.EditPageData{
			Error: "Error counting users",
		})
		return
	}
	totalPages := int((total + int64(userPageLimit) - 1) / int64(userPageLimit))

	flash := utils.GetFlashMessage(w, r)

	render.RenderTemplateWithData(w, "Home.html", model.EditPageData{
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
