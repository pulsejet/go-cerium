package models

import (
	"time"
)

type Form struct {
	Name    string `json:"name"`
	Creator string `json:"creator"`
	Pages   []Page `json:"pages"`
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
	Responses map[string]interface{} `json:"responses"`
}
