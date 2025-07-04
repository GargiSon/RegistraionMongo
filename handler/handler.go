package handler

import (
	"context"
	"go2/model"
	"go2/mongo"
	"go2/render"
	"go2/utils"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	setNoCacheHeaders(w)

	page := 1

	email, ok := GetSessionEmail(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	adminName := ""
	if parts := strings.Split(email, "@"); len(parts) > 0 {
		adminName = parts[0]
	}

	// Get query parameters
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
		render.RenderTemplateWithData(w, "Home.html", model.HomePageData{
			Error: "Error counting users",
		})
		return
	}

	totalPages := int((total + int64(userPageLimit) - 1) / int64(userPageLimit))
	flash := utils.GetFlashMessage(w, r)

	render.RenderTemplateWithData(w, "Home.html", model.HomePageData{
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
