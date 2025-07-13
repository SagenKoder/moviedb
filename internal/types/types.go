package types

import "time"

type User struct {
	ID        int       `json:"id"`
	Auth0ID   string    `json:"auth0_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Username  *string   `json:"username"`
	AvatarURL *string   `json:"avatar_url"`
	Created   time.Time `json:"created_at"`
}

type Movie struct {
	ID        int       `json:"id"`
	TMDBID    int       `json:"tmdb_id"`
	Title     string    `json:"title"`
	Year      *int      `json:"year"`
	PosterURL *string   `json:"poster_url"`
	Synopsis  *string   `json:"synopsis"`
	Runtime   *int      `json:"runtime"`
	Genres    *string   `json:"genres"` // JSON string
	Created   time.Time `json:"created_at"`
}

type UserMovie struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	MovieID      int        `json:"movie_id"`
	Status       string     `json:"status"`
	Rating       *int       `json:"rating"`
	WatchedDate  *time.Time `json:"watched_date"`
	Notes        *string    `json:"notes"`
	OwnedFormats *string    `json:"owned_formats"` // JSON string
	Created      time.Time  `json:"created_at"`
	Updated      time.Time  `json:"updated_at"`
}

type List struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsPublic    bool      `json:"is_public"`
	Created     time.Time `json:"created_at"`
}

type ListMovie struct {
	ID      int       `json:"id"`
	ListID  int       `json:"list_id"`
	MovieID int       `json:"movie_id"`
	Added   time.Time `json:"added_at"`
}

type Friend struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	FriendID int       `json:"friend_id"`
	Created  time.Time `json:"created_at"`
}

type FeedPost struct {
	ID       int        `json:"id"`
	UserID   int        `json:"user_id"`
	Type     string     `json:"type"`
	MovieID  *int       `json:"movie_id"`
	ListID   *int       `json:"list_id"`
	Content  *string    `json:"content"`
	Rating   *int       `json:"rating"`
	Metadata *string    `json:"metadata"` // JSON string
	Created  time.Time  `json:"created_at"`
}

type PostLike struct {
	ID      int       `json:"id"`
	PostID  int       `json:"post_id"`
	UserID  int       `json:"user_id"`
	Created time.Time `json:"created_at"`
}

type PostComment struct {
	ID      int       `json:"id"`
	PostID  int       `json:"post_id"`
	UserID  int       `json:"user_id"`
	Content string    `json:"content"`
	Created time.Time `json:"created_at"`
}

// Request/Response types
type UpdateMovieStatusRequest struct {
	Status string `json:"status"`
}

type RateMovieRequest struct {
	Rating int `json:"rating"`
}

type UpdateNotesRequest struct {
	Notes string `json:"notes"`
}

type UpdateOwnedFormatsRequest struct {
	Formats []string `json:"formats"`
}

type CreateListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

type UpdateListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

type AddCommentRequest struct {
	Content string `json:"content"`
}

type UserPreferences struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	DarkMode bool      `json:"dark_mode"`
	Created  time.Time `json:"created_at"`
	Updated  time.Time `json:"updated_at"`
}

type UpdatePreferencesRequest struct {
	DarkMode bool `json:"darkMode"`
}