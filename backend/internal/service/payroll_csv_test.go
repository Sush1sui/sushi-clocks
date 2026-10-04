package service

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sushi-clocks/backend/internal/domain"
)

func TestBuildPayrollCSV(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

	preview := &domain.PayrollPreview{
		CompanyID:    "test-comp-1",
		CurrencyCode: "USD",
		PeriodStart:  start,
		PeriodEnd:    end,
		TotalGross:   2000.00,
		TotalNet:     1900.00,
		Employees: []domain.EmployeePayrollLine{
			{
				UserID:          "usr-1",
				FirstName:       "Alice",
				LastName:        "Smith",
				Email:           "alice@example.com",
				RoleName:        "Chef",
				CurrencyCode:    "USD",
				WageType:        "hourly",
				BaseRate:        25.00,
				WageSource:      "role",
				WorkedMinutes:   4800, // 80 hrs
				WorkedDays:      10,
				PaidLeaveDays:   1,
				UnpaidLeaveDays: 0,
				GrossPay:        2000.00,
				Additions:       0.00,
				Deductions:      100.00,
				NetPay:          1900.00,
				Modifiers: []domain.PayrollModifier{
					{
						ID:                "mod-1",
						Name:              "Health Insurance",
						ModifierType:      "deduction",
						CalculationMethod: "fixed",
						Value:             100.00,
						Amount:            100.00,
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := BuildPayrollCSV(&buf, preview)
	if err != nil {
		t.Fatalf("BuildPayrollCSV error: %v", err)
	}

	csvStr := buf.String()
	lines := strings.Split(strings.TrimSpace(csvStr), "\r\n")
	if len(lines) < 2 {
		lines = strings.Split(strings.TrimSpace(csvStr), "\n")
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 row), got %d: %v", len(lines), lines)
	}

	headerCols := strings.Split(lines[0], ",")
	if len(headerCols) != 19 {
		t.Errorf("expected 19 columns in header, got %d", len(headerCols))
	}

	if !strings.Contains(lines[1], "Alice") || !strings.Contains(lines[1], "Chef") {
		t.Errorf("expected row to contain Alice and Chef, got %s", lines[1])
	}
}

func TestPayrollFilename(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	name := PayrollFilename(start)
	expected := "payroll_2026-10-01.csv"
	if name != expected {
		t.Errorf("expected %s, got %s", expected, name)
	}
}

func TestSanitizeCSVCell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"=CMD|' /C calc'!A0", "'=CMD|' /C calc'!A0"},
		{"+12345", "'+12345"},
		{"-formula", "'-formula"},
		{"@SUM(A1:A10)", "'@SUM(A1:A10)"},
		{"\tmalicious", "'\tmalicious"},
		{"\rbreak", "'\rbreak"},
		{"", ""},
	}

	for _, tc := range tests {
		got := SanitizeCSVCell(tc.input)
		if got != tc.expected {
			t.Errorf("SanitizeCSVCell(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}
