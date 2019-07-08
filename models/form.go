package models

import (
	"time"
)

type Form struct {
	Name           string `json:"name"`
	Creator        string `json:"creator"`
	CanEdit        bool   `json:"can_edit"`
	Pages          []Page `json:"pages"`
	RequireLogin   bool   `json:"require_login"`
	CollectEmail   bool   `json:"collect_email"`
	SingleResponse bool   `json:"single_response"`
}

type Page struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Widgets     []Widget `json:"widgets"`
}

type Widget struct {
	Type  string                 `json:"type"`
	Uid   string                 `json:"uid"`
	Props map[string]interface{} `json:"props"`
}

type FormResponse struct {
	FormId    string                 `json:"form_id"`
	Timestamp time.Time              `json:"timestamp"`
	Filler    string                 `json:"filler"`
	Responses map[string]interface{} `json:"responses"`
}

type FormAnonResponder struct {
	FormId string
	Filler string
}
