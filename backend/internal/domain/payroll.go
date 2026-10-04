package domain

import "time"

const (
	WageTypeHourly  = "hourly"
	WageTypeDaily   = "daily"
	WageTypeMonthly = "monthly"

	WageSourceOverride = "override"
	WageSourceRole     = "role"
	WageSourceNone     = "none"

	ModifierTypeAddition  = "addition"
	ModifierTypeDeduction = "deduction"

	CalculationMethodFixed      = "fixed"
	CalculationMethodPercentage = "percentage"
)

type ResolvedWage struct {
	UserID   string  `json:"user_id"`
	WageType string  `json:"wage_type"` // "hourly" | "daily" | "monthly"
	Rate     float64 `json:"rate"`
	Source   string  `json:"source"` // "override" | "role"
}

type PayrollModifier struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	ModifierType      string  `json:"modifier_type"`      // "addition" | "deduction"
	CalculationMethod string  `json:"calculation_method"` // "fixed" | "percentage"
	Value             float64 `json:"value"`
	Amount            float64 `json:"amount"` // evaluated monetary amount
}

type EmployeePayrollLine struct {
	UserID          string            `json:"user_id"`
	FirstName       string            `json:"first_name"`
	LastName        string            `json:"last_name"`
	Email           string            `json:"email"`
	RoleName        string            `json:"role_name"`
	CurrencyCode    string            `json:"currency_code"`
	WageType        string            `json:"wage_type"`
	BaseRate        float64           `json:"base_rate"`
	WageSource      string            `json:"wage_source"` // "override" | "role" | "none"
	WorkedMinutes   float64           `json:"worked_minutes"`
	WorkedDays      float64           `json:"worked_days"`
	PaidLeaveDays   int               `json:"paid_leave_days"`
	UnpaidLeaveDays int               `json:"unpaid_leave_days"`
	GrossPay        float64           `json:"gross_pay"`
	Additions       float64           `json:"additions"`
	Deductions      float64           `json:"deductions"`
	NetPay          float64           `json:"net_pay"`
	Modifiers       []PayrollModifier `json:"modifiers"`
}

type PayrollPreview struct {
	CompanyID    string                `json:"company_id"`
	CurrencyCode string                `json:"currency_code"`
	PeriodStart  time.Time             `json:"period_start"`
	PeriodEnd    time.Time             `json:"period_end"`
	Employees    []EmployeePayrollLine `json:"employees"`
	TotalGross   float64               `json:"total_gross"`
	TotalNet     float64               `json:"total_net"`
}
