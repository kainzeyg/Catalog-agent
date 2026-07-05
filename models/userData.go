package models

type UserData struct {
	Username string   `json:"username"`
	UPN      string   `json:"upn"`
	Groups   []string `json:"groups"`
	Domain   string   `json:"domain"`
}
