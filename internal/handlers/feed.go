package handlers

import (
	"database/sql"
	"net/http"
)

type FeedHandler struct {
	db *sql.DB
}

func NewFeedHandler(db *sql.DB) *FeedHandler {
	return &FeedHandler{db: db}
}

func (h *FeedHandler) GetFriendsFeed(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement friends feed
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *FeedHandler) GetGlobalFeed(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement global feed
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *FeedHandler) LikePost(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement like post
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *FeedHandler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement unlike post
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *FeedHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement add comment
	w.WriteHeader(http.StatusNotImplemented)
}