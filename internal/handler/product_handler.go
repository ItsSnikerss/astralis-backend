package handler

import (
	"astralis.backend/internal/database"
	"astralis.backend/internal/model"
	"encoding/json"
	"net/http"
)

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query("SELECT id, name, description, price, is_featured, sort_index FROM products ORDER BY sort_index ASC")
	if err != nil {
		http.Error(w, `{"error":"Failed to query products"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	products := make([]model.Product, 0)
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.IsFeatured, &p.SortIndex); err != nil {
			http.Error(w, `{"error":"Failed to scan product row"}`, http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}