package domain

import "time"

type CompanyWithStats struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CurrencyCode string    `json:"currency_code"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
	TotalUsers   int       `json:"total_users"`
	AdminEmail   *string   `json:"admin_email,omitempty"`
}

type CreateCompanyRequest struct {
	Name           string `json:"name"`
	CurrencyCode   string `json:"currency_code"`
	Timezone       string `json:"timezone"`
	AdminFirstName string `json:"admin_first_name"`
	AdminLastName  string `json:"admin_last_name"`
	AdminEmail     string `json:"admin_email"`
	AdminPassword  string `json:"admin_password"`
}
