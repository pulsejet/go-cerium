package models

import (
	"time"
)

// Form : a single form stored in database
type Form struct {
	Name           string    `json:"name"`
	Creator        string    `json:"creator"`
	Timestamp      time.Time `json:"timestamp"`
	CanEdit        bool      `json:"can_edit"`
	Pages          []Page    `json:"pages"`
	RequireLogin   bool      `json:"require_login"`
	CollectEmail   bool      `json:"collect_email"`
	SingleResponse bool      `json:"single_response"`
	IsClosed       bool      `json:"is_closed"`
	ResponseToken  string    `json:"-"`
}

// Page : a section in  a form
type Page struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Widgets     []Widget `json:"widgets"`
}

// Widget : a single control in a page
type Widget struct {
	Type  string                 `json:"type"`
	UID   string                 `json:"uid"`
	Props map[string]interface{} `json:"props"`
}

// FormResponse : a single response to a form
type FormResponse struct {
	FormID    string                 `json:"form_id"`
	Timestamp time.Time              `json:"timestamp"`
	Filler    string                 `json:"filler"`
	Responses map[string]interface{} `json:"responses"`
}

// FormAnonResponder : mapping between form and filler for singe-response
type FormAnonResponder struct {
	FormID string
	Filler string
}
