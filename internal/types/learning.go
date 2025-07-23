package types

import "time"

// Course represents a LinkedIn Learning course
type Course struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Instructor  string    `json:"instructor"`
	Duration    string    `json:"duration"`
	Level       string    `json:"level"`
	Chapters    []Chapter `json:"chapters"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Chapter represents a course chapter
type Chapter struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Duration string    `json:"duration"`
	Order    int       `json:"order"`
	Sections []Section `json:"sections"`
}

// Section represents a chapter section/lesson
type Section struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Duration string `json:"duration"`
	Order    int    `json:"order"`
	URL      string `json:"url"`
}

// ScrapingResult represents the result of a scraping operation
type ScrapingResult struct {
	Success bool   `json:"success"`
	Course  Course `json:"course,omitempty"`
	Error   string `json:"error,omitempty"`
}