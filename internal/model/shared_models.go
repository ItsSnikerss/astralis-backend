package model

import "time"

type UserForAdmin struct {
	ID                    int        `json:"id"`
	Username              string     `json:"username"`
	Email                 string     `json:"email"`
	Role                  string     `json:"role"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at"`
	IsBanned              bool       `json:"is_banned"`
	Hwid                  *string    `json:"hwid"`
}

type KeyForAdmin struct {
	ID           int        `json:"id"`
	KeyString    string     `json:"key_string"`
	DurationDays int        `json:"duration_days"`
	IsUsed       bool       `json:"is_used"`
	UsedByUserID *int       `json:"used_by_user_id"`
	UsedAt       *time.Time `json:"used_at"`
}

type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	IsFeatured  bool   `json:"is_featured"`
	SortIndex   int    `json:"sort_index"`
}

type PaginatedUsersResponse struct {
	Users       []UserForAdmin `json:"users"`
	TotalPages  int            `json:"total_pages"`
	CurrentPage int            `json:"current_page"`
}

type PaginatedKeysResponse struct {
	Keys        []KeyForAdmin `json:"keys"`
	TotalPages  int           `json:"total_pages"`
	CurrentPage int           `json:"current_page"`
}