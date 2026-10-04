package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/repository"
)

type ArchiveService struct {
	telemetryRepo *repository.TelemetryRepository
	auditRepo     *repository.AuditRepository
	userRepo      *repository.UserRepository
	emailService  *EmailService
}

func NewArchiveService(
	telemetryRepo *repository.TelemetryRepository,
	auditRepo *repository.AuditRepository,
	userRepo *repository.UserRepository,
	emailService *EmailService,
) *ArchiveService {
	return &ArchiveService{
		telemetryRepo: telemetryRepo,
		auditRepo:     auditRepo,
		userRepo:      userRepo,
		emailService:  emailService,
	}
}

// BuildTelemetryCSV generates an RFC 4180 CSV with OWASP formula injection protection.
func (s *ArchiveService) BuildTelemetryCSV(records []domain.PunchTelemetry) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.UseCRLF = true

	headers := []string{"ID", "Company ID", "User ID", "Action", "Timestamp", "IP", "Device", "OS", "Browser"}
	if err := w.Write(headers); err != nil {
		return nil, err
	}

	for _, r := range records {
		row := []string{
			SanitizeCSVCell(r.ID),
			SanitizeCSVCell(r.CompanyID),
			SanitizeCSVCell(r.UserID),
			SanitizeCSVCell(r.Action),
			r.Timestamp.UTC().Format(time.RFC3339),
			SanitizeCSVCell(r.IP),
			SanitizeCSVCell(r.Device),
			SanitizeCSVCell(r.OS),
			SanitizeCSVCell(r.Browser),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// BuildAuditCSV generates an RFC 4180 CSV with OWASP formula injection protection.
func (s *ArchiveService) BuildAuditCSV(records []domain.AuditLog) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.UseCRLF = true

	headers := []string{"ID", "Company ID", "Collection", "Action", "Actor ID", "Target ID", "Reason", "Timestamp", "Before", "After"}
	if err := w.Write(headers); err != nil {
		return nil, err
	}

	for _, r := range records {
		beforeJSON, _ := json.Marshal(r.Before)
		afterJSON, _ := json.Marshal(r.After)

		row := []string{
			SanitizeCSVCell(r.ID),
			SanitizeCSVCell(r.CompanyID),
			SanitizeCSVCell(r.Collection),
			SanitizeCSVCell(r.Action),
			SanitizeCSVCell(r.ActorID),
			SanitizeCSVCell(r.TargetID),
			SanitizeCSVCell(r.Reason),
			r.Timestamp.UTC().Format(time.RFC3339),
			SanitizeCSVCell(string(beforeJSON)),
			SanitizeCSVCell(string(afterJSON)),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// RunArchival processes records older than cutoffDays (default 365), sends CSV emails to admins/opt-in HR,
// and removes the purged records from MongoDB Atlas.
func (s *ArchiveService) RunArchival(ctx context.Context, cutoffDays int) error {
	if s.telemetryRepo == nil && s.auditRepo == nil {
		return nil
	}
	if cutoffDays <= 0 {
		cutoffDays = 365
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -cutoffDays)
	log.Printf("[ArchiveService] Starting archival run for records older than %s (>= %d days)", cutoff.Format("2006-01-02"), cutoffDays)

	companyTelemetries := make(map[string][]domain.PunchTelemetry)
	if s.telemetryRepo != nil {
		telemetryRecords, err := s.telemetryRepo.GetRecordsForArchival(ctx, cutoff)
		if err != nil {
			log.Printf("[ArchiveService] error fetching telemetry: %v", err)
		} else {
			for _, rec := range telemetryRecords {
				companyTelemetries[rec.CompanyID] = append(companyTelemetries[rec.CompanyID], rec)
			}
		}
	}

	companyAudits := make(map[string][]domain.AuditLog)
	if s.auditRepo != nil {
		auditRecords, err := s.auditRepo.GetRecordsForArchival(ctx, cutoff)
		if err != nil {
			log.Printf("[ArchiveService] error fetching audit logs: %v", err)
		} else {
			for _, rec := range auditRecords {
				companyAudits[rec.CompanyID] = append(companyAudits[rec.CompanyID], rec)
			}
		}
	}

	// Gather all unique company IDs that have expiring records
	companySet := make(map[string]struct{})
	for cid := range companyTelemetries {
		companySet[cid] = struct{}{}
	}
	for cid := range companyAudits {
		companySet[cid] = struct{}{}
	}

	if len(companySet) == 0 {
		log.Println("[ArchiveService] No records older than 1 year found to archive")
		return nil
	}

	dateStr := cutoff.Format("2006-01-02")
	for companyID := range companySet {
		recipients, err := s.userRepo.GetArchiveRecipientsForCompany(ctx, companyID)
		if err != nil {
			log.Printf("[ArchiveService] error getting recipients for company %s: %v", companyID, err)
			continue
		}

		if len(recipients) == 0 {
			log.Printf("[ArchiveService] no eligible admin or opted-in HR recipients found for company %s", companyID)
			continue
		}

		toEmails := make([]string, 0, len(recipients))
		for _, u := range recipients {
			if u.Email != "" {
				toEmails = append(toEmails, u.Email)
			}
		}

		var attachments []EmailAttachment

		// 1. Telemetry attachment
		if tRecs := companyTelemetries[companyID]; len(tRecs) > 0 {
			data, err := s.BuildTelemetryCSV(tRecs)
			if err == nil {
				attachments = append(attachments, EmailAttachment{
					Filename:    fmt.Sprintf("punch_telemetry_archive_up_to_%s.csv", dateStr),
					ContentType: "text/csv",
					Data:        data,
				})
			}
		}

		// 2. Audit log attachment
		if aRecs := companyAudits[companyID]; len(aRecs) > 0 {
			data, err := s.BuildAuditCSV(aRecs)
			if err == nil {
				attachments = append(attachments, EmailAttachment{
					Filename:    fmt.Sprintf("audit_logs_archive_up_to_%s.csv", dateStr),
					ContentType: "text/csv",
					Data:        data,
				})
			}
		}

		if len(attachments) == 0 {
			continue
		}

		subject := fmt.Sprintf("[Sushi-Clocks] 1-Year Audit & Telemetry Archive Export (%s)", dateStr)
		body := fmt.Sprintf(`
			<h2>Sushi-Clocks System Data Archival</h2>
			<p>Hello,</p>
			<p>In accordance with data retention policies, records older than 1 year up to <strong>%s</strong> have been archived from active database storage.</p>
			<p>Attached to this email are the permanent CSV records for your company:</p>
			<ul>
				<li>Punch clock-in/out device and network telemetry logs</li>
				<li>Timesheet adjustments, overrides, and leave request audit logs</li>
			</ul>
			<p>Please store these attached CSV files safely for compliance.</p>
			<hr>
			<p><em>Sushi-Clocks Automated Compliance & Archival Engine</em></p>
		`, dateStr)

		if err := s.emailService.SendWithAttachments(toEmails, subject, body, attachments); err != nil {
			log.Printf("[ArchiveService] failed to send email for company %s: %v", companyID, err)
			// Do not delete records if email delivery failed
			continue
		}

		// Email sent successfully: purge archived records from MongoDB
		if s.telemetryRepo != nil {
			delT, err := s.telemetryRepo.DeleteArchivedRecords(ctx, companyID, cutoff)
			if err != nil {
				log.Printf("[ArchiveService] error deleting telemetry for company %s: %v", companyID, err)
			} else {
				log.Printf("[ArchiveService] purged %d telemetry records for company %s", delT, companyID)
			}
		}
		if s.auditRepo != nil {
			delA, err := s.auditRepo.DeleteArchivedRecords(ctx, companyID, cutoff)
			if err != nil {
				log.Printf("[ArchiveService] error deleting audit logs for company %s: %v", companyID, err)
			} else {
				log.Printf("[ArchiveService] purged %d audit records for company %s", delA, companyID)
			}
		}
	}

	log.Println("[ArchiveService] Archival run completed successfully")
	return nil
}

// StartScheduler runs RunArchival periodically (e.g. daily) in a background goroutine.
func (s *ArchiveService) StartScheduler(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.RunArchival(ctx, 365); err != nil {
					log.Printf("[ArchiveService] periodic archival error: %v", err)
				}
			}
		}
	}()
}
