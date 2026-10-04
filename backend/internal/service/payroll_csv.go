package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/sushi-clocks/backend/internal/domain"
)

// PayrollFilename returns standard CSV filename based on pay period start.
func PayrollFilename(periodStart time.Time) string {
	return fmt.Sprintf("payroll_%s.csv", periodStart.Format("2006-01-02"))
}

// SanitizeCSVCell prevents CSV Formula Injection (OWASP A03)
// by prepending a single quote if the field begins with =, +, -, @, \t, or \r.
func SanitizeCSVCell(val string) string {
	if len(val) == 0 {
		return val
	}
	switch val[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + val
	default:
		return val
	}
}

// BuildPayrollCSV writes the 19-column RFC 4180 compliant CSV to the provided writer.
func BuildPayrollCSV(w io.Writer, preview *domain.PayrollPreview) error {
	writer := csv.NewWriter(w)
	writer.UseCRLF = true

	headers := []string{
		"User ID",
		"First Name",
		"Last Name",
		"Email",
		"Role",
		"Currency",
		"Wage Type",
		"Base Rate",
		"Wage Source",
		"Worked Minutes",
		"Worked Hours",
		"Worked Days",
		"Paid Leave Days",
		"Unpaid Leave Days",
		"Gross Pay",
		"Additions",
		"Deductions",
		"Net Pay",
		"Pay Period",
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write csv header error: %w", err)
	}

	periodStr := fmt.Sprintf("%s to %s",
		preview.PeriodStart.Format("2006-01-02"),
		preview.PeriodEnd.Format("2006-01-02"),
	)

	for _, emp := range preview.Employees {
		workedHours := emp.WorkedMinutes / 60.0

		row := []string{
			SanitizeCSVCell(emp.UserID),
			SanitizeCSVCell(emp.FirstName),
			SanitizeCSVCell(emp.LastName),
			SanitizeCSVCell(emp.Email),
			SanitizeCSVCell(emp.RoleName),
			SanitizeCSVCell(emp.CurrencyCode),
			SanitizeCSVCell(emp.WageType),
			fmt.Sprintf("%.2f", emp.BaseRate),
			SanitizeCSVCell(emp.WageSource),
			fmt.Sprintf("%.2f", emp.WorkedMinutes),
			fmt.Sprintf("%.2f", workedHours),
			fmt.Sprintf("%.2f", emp.WorkedDays),
			fmt.Sprintf("%d", emp.PaidLeaveDays),
			fmt.Sprintf("%d", emp.UnpaidLeaveDays),
			fmt.Sprintf("%.2f", emp.GrossPay),
			fmt.Sprintf("%.2f", emp.Additions),
			fmt.Sprintf("%.2f", emp.Deductions),
			fmt.Sprintf("%.2f", emp.NetPay),
			periodStr,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row error: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}
