package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
	"github.com/sushi-clocks/backend/internal/utils"
)

type PayrollService struct {
	timesheetRepo *repository.TimesheetRepository
	leaveRepo     *repository.LeaveRepository
	wageRepo      *repository.WageRepository
	companyRepo   *repository.CompanyRepository
	userRepo      *repository.UserRepository
}

func NewPayrollService(
	timesheetRepo *repository.TimesheetRepository,
	leaveRepo     *repository.LeaveRepository,
	wageRepo      *repository.WageRepository,
	companyRepo   *repository.CompanyRepository,
	userRepo      *repository.UserRepository,
) *PayrollService {
	return &PayrollService{
		timesheetRepo: timesheetRepo,
		leaveRepo:     leaveRepo,
		wageRepo:      wageRepo,
		companyRepo:   companyRepo,
		userRepo:      userRepo,
	}
}

// Calculate computes the full payroll preview for all non-super-admin employees in the company
// over the given period [start, end].
func (s *PayrollService) Calculate(ctx context.Context, companyID string, start, end time.Time) (*domain.PayrollPreview, error) {
	// 1. Fetch company configuration
	company, err := s.companyRepo.GetCompanyByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch company: %w", err)
	}

	tz := company.Timezone
	loc := time.UTC
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}

	// Normalise period boundaries
	pStartDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	pEndDay := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, loc)
	periodStartDayOnly := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	periodEndDayOnly := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)

	// 2. Fetch all employees in company
	users, err := s.userRepo.GetUsersByCompanyID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	// 3. Batch resolve wages
	wageMap, err := s.wageRepo.ResolveAllUserWages(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve wages: %w", err)
	}

	// 4. Batch fetch completed timesheets in period
	timesheetMap, err := s.timesheetRepo.GetCompanyTimesheetsForPayroll(ctx, companyID, pStartDay, pEndDay)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timesheets: %w", err)
	}

	// 5. Batch fetch approved leaves in period
	leaveMap, err := s.leaveRepo.GetCompanyApprovedLeavesForPayroll(ctx, companyID, pStartDay, pEndDay)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leaves: %w", err)
	}

	// 6. Batch fetch all user modifiers in company (O(1) batch query)
	allModifiersMap, err := s.wageRepo.GetAllCompanyUserModifiers(ctx, companyID)
	if err != nil {
		allModifiersMap = make(map[string][]domain.PayrollModifier)
	}

	// 7. Batch fetch all user role names in company (O(1) batch query)
	allRoleNamesMap, err := s.wageRepo.GetAllCompanyUserRoleNames(ctx, companyID)
	if err != nil {
		allRoleNamesMap = make(map[string]string)
	}

	lines := make([]domain.EmployeePayrollLine, 0, len(users))
	var totalGross, totalNet float64

	for _, user := range users {
		// A. Aggregate worked time from completed shifts
		var workedMinutes float64
		uniqueWorkedDays := make(map[string]bool)

		userTimesheets := timesheetMap[user.ID]
		for _, ts := range userTimesheets {
			if ts.ClockOutTime == nil {
				continue
			}
			segments, err := utils.SplitShiftByCalendarDay(ts.ClockInTime, *ts.ClockOutTime, tz)
			if err != nil {
				continue
			}
			for _, seg := range segments {
				segDay := time.Date(seg.LocalDate.Year(), seg.LocalDate.Month(), seg.LocalDate.Day(), 0, 0, 0, 0, loc)
				if !segDay.Before(periodStartDayOnly) && !segDay.After(periodEndDayOnly) {
					workedMinutes += seg.MinutesWorked
					uniqueWorkedDays[segDay.Format("2006-01-02")] = true
				}
			}
		}
		workedDays := float64(len(uniqueWorkedDays))

		// B. Aggregate leaves clipped to period
		var paidLeaveDays, unpaidLeaveDays int
		userLeaves := leaveMap[user.ID]
		for _, l := range userLeaves {
			days := utils.ClipLeaveDaysToPeriod(l.StartDate, l.EndDate, start, end)
			if l.IsPaid {
				paidLeaveDays += days
			} else {
				unpaidLeaveDays += days
			}
		}

		// C. Determine wage rate and source
		wageType := domain.WageTypeHourly
		baseRate := 0.0
		wageSource := domain.WageSourceNone

		if rw, ok := wageMap[user.ID]; ok {
			wageType = rw.WageType
			baseRate = rw.Rate
			wageSource = rw.Source
		}

		// D. Compute gross pay based on wage_type
		var gross float64
		if baseRate > 0 && wageSource != domain.WageSourceNone {
			switch wageType {
			case domain.WageTypeHourly:
				payableHours := (workedMinutes / 60.0) + float64(paidLeaveDays*8) - float64(unpaidLeaveDays*8)
				if payableHours < 0 {
					payableHours = 0
				}
				gross = payableHours * baseRate

			case domain.WageTypeDaily:
				payableDays := workedDays + float64(paidLeaveDays) - float64(unpaidLeaveDays)
				if payableDays < 0 {
					payableDays = 0
				}
				gross = payableDays * baseRate

			case domain.WageTypeMonthly:
				effectiveDays := int(math.Round(workedDays)) + paidLeaveDays - unpaidLeaveDays
				if effectiveDays < 0 {
					effectiveDays = 0
				}
				prorated, err := utils.ProrateMonthlyWage(
					baseRate,
					company.ProrationType,
					start,
					end,
					effectiveDays,
					company.WorkDaysMask,
				)
				if err == nil {
					gross = prorated
				}
			}
		}
		gross = math.Round(gross*100) / 100

		// E. Modifiers (O(1) in-memory lookup from batch query)
		modifiers := allModifiersMap[user.ID]

		var additions, deductions float64
		evaluatedModifiers := make([]domain.PayrollModifier, 0, len(modifiers))
		for _, m := range modifiers {
			var amount float64
			if m.CalculationMethod == domain.CalculationMethodPercentage {
				amount = math.Round((gross*(m.Value/100.0))*100) / 100
			} else {
				amount = math.Round(m.Value*100) / 100
			}
			m.Amount = amount

			if m.ModifierType == domain.ModifierTypeAddition {
				additions += amount
			} else if m.ModifierType == domain.ModifierTypeDeduction {
				deductions += amount
			}
			evaluatedModifiers = append(evaluatedModifiers, m)
		}

		additions = math.Round(additions*100) / 100
		deductions = math.Round(deductions*100) / 100

		net := gross + additions - deductions
		if net < 0 {
			net = 0
		}
		net = math.Round(net*100) / 100

		// F. Resolve role name (O(1) in-memory lookup from batch query)
		roleName := allRoleNamesMap[user.ID]
		if roleName == "" {
			roleName = user.SystemRole
		}

		line := domain.EmployeePayrollLine{
			UserID:          user.ID,
			FirstName:       user.FirstName,
			LastName:        user.LastName,
			Email:           user.Email,
			RoleName:        roleName,
			CurrencyCode:    company.CurrencyCode,
			WageType:        wageType,
			BaseRate:        baseRate,
			WageSource:      wageSource,
			WorkedMinutes:   math.Round(workedMinutes*100) / 100,
			WorkedDays:      workedDays,
			PaidLeaveDays:   paidLeaveDays,
			UnpaidLeaveDays: unpaidLeaveDays,
			GrossPay:        gross,
			Additions:       additions,
			Deductions:      deductions,
			NetPay:          net,
			Modifiers:       evaluatedModifiers,
		}

		lines = append(lines, line)
		totalGross += gross
		totalNet += net
	}

	preview := &domain.PayrollPreview{
		CompanyID:    company.ID,
		CurrencyCode: company.CurrencyCode,
		PeriodStart:  start,
		PeriodEnd:    end,
		Employees:    lines,
		TotalGross:   math.Round(totalGross*100) / 100,
		TotalNet:     math.Round(totalNet*100) / 100,
	}

	return preview, nil
}
