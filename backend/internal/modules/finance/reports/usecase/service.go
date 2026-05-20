package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"

	"pekan/backend/internal/modules/finance/reports/domain"
	"pekan/backend/internal/platform/storage"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo       domain.Repository
	exportRepo domain.ExportRepository
	authz      Authorizer
	audit      AuditLogger
	storage    storage.ObjectStorage
}

func NewService(repo domain.Repository, exportRepo domain.ExportRepository, authz Authorizer, audit AuditLogger, storageProvider storage.ObjectStorage) *Service {
	return &Service{
		repo:       repo,
		exportRepo: exportRepo,
		authz:      authz,
		audit:      audit,
		storage:    storageProvider,
	}
}

type CreateTransactionsReportInput struct {
	TenantID    string
	ActorUserID string
	ReportType  string
	DateFrom    *string
	DateTo      *string
	CategoryID  *string
	Type        *string
	Status      *string
	Format      string
}

func (s *Service) CreateTransactionsReport(ctx context.Context, in CreateTransactionsReportInput) (domain.Report, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reports"); err != nil {
		return domain.Report{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reports.write"); err != nil {
		return domain.Report{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reports.create"); err != nil {
		return domain.Report{}, err
	}

	reportType := normalizeReportType(in.ReportType)
	if !isSupportedReportType(reportType) {
		return domain.Report{}, domain.ErrInvalidReportType
	}
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if !isSupportedFormat(format) {
		return domain.Report{}, domain.ErrInvalidFormat
	}

	params, _ := json.Marshal(map[string]any{
		"report_type": reportType,
		"date_from":   in.DateFrom,
		"date_to":     in.DateTo,
		"category_id": in.CategoryID,
		"type":        in.Type,
		"status":      in.Status,
	})

	now := time.Now().UTC()
	report := domain.Report{
		TenantID:   in.TenantID,
		ReportType: reportType,
		Format:     format,
		Status:     "queued",
		Params:     params,
		CreatedBy:  in.ActorUserID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	created, err := s.repo.Create(ctx, report)
	if err != nil {
		return domain.Report{}, err
	}

	contentType, body, err := s.generateContent(ctx, in, reportType, format)
	if err != nil {
		created.Status = "failed"
		created.UpdatedAt = time.Now().UTC()
		_, _ = s.repo.UpdateStatus(ctx, created)
		return domain.Report{}, err
	}

	objectKey := fmt.Sprintf("reports/%s/%s/%s.%s", in.TenantID, reportType, created.ID, format)
	putOut, err := s.storage.Put(ctx, storage.PutObjectInput{
		TenantID:    in.TenantID,
		Module:      "finance.reports",
		ObjectKey:   objectKey,
		ContentType: contentType,
		Body:        body,
	})
	if err != nil {
		created.Status = "failed"
		created.UpdatedAt = time.Now().UTC()
		_, _ = s.repo.UpdateStatus(ctx, created)
		return domain.Report{}, err
	}

	created.Status = "ready"
	created.StorageProvider = &putOut.Provider
	created.StorageKey = &putOut.ObjectKey
	created.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.UpdateStatus(ctx, created)
	if err != nil {
		return domain.Report{}, err
	}

	_ = s.audit.Write(ctx, "finance.report.create", "finance_report", updated.ID, nil, map[string]any{
		"id":          updated.ID,
		"report_type": reportType,
		"format":      format,
		"status":      updated.Status,
	})
	return updated, nil
}

func (s *Service) generateContent(ctx context.Context, in CreateTransactionsReportInput, reportType, format string) (string, io.Reader, error) {
	switch reportType {
	case "transactions":
		rows, err := s.exportRepo.ListTransactions(ctx, in.TenantID, in.DateFrom, in.DateTo, in.CategoryID, in.Type)
		if err != nil {
			return "", nil, err
		}
		if format == "csv" {
			return "text/csv", buildTransactionsCSV(rows), nil
		}
		pdfBytes, err := renderTransactionsPDF(rows)
		if err != nil {
			return "", nil, err
		}
		return "application/pdf", bytes.NewReader(pdfBytes), nil
	case "savings":
		rows, err := s.exportRepo.ListSavings(ctx, in.TenantID, in.DateFrom, in.DateTo, in.Status)
		if err != nil {
			return "", nil, err
		}
		if format == "csv" {
			return "text/csv", buildSavingsCSV(rows), nil
		}
		pdfBytes, err := renderSavingsPDF(rows)
		if err != nil {
			return "", nil, err
		}
		return "application/pdf", bytes.NewReader(pdfBytes), nil
	case "budgets":
		rows, err := s.exportRepo.ListBudgets(ctx, in.TenantID, in.DateFrom, in.DateTo, in.Status)
		if err != nil {
			return "", nil, err
		}
		if format == "csv" {
			return "text/csv", buildBudgetsCSV(rows), nil
		}
		pdfBytes, err := renderBudgetsPDF(rows)
		if err != nil {
			return "", nil, err
		}
		return "application/pdf", bytes.NewReader(pdfBytes), nil
	case "reminders":
		rows, err := s.exportRepo.ListReminders(ctx, in.TenantID, in.DateFrom, in.DateTo, in.Status)
		if err != nil {
			return "", nil, err
		}
		if format == "csv" {
			return "text/csv", buildRemindersCSV(rows), nil
		}
		pdfBytes, err := renderRemindersPDF(rows)
		if err != nil {
			return "", nil, err
		}
		return "application/pdf", bytes.NewReader(pdfBytes), nil
	default:
		return "", nil, domain.ErrInvalidReportType
	}
}

func (s *Service) GetByID(ctx context.Context, tenantID, reportID string) (domain.Report, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reports"); err != nil {
		return domain.Report{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reports.read"); err != nil {
		return domain.Report{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reports.read"); err != nil {
		return domain.Report{}, err
	}
	return s.repo.GetByID(ctx, tenantID, reportID)
}

type ListInput struct {
	TenantID string
	Page     int
	PageSize int
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Report, int64, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reports"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reports.read"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reports.read"); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, domain.ListFilter{
		TenantID: in.TenantID,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
}

func (s *Service) Download(ctx context.Context, tenantID, reportID string) (domain.Report, io.ReadCloser, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reports"); err != nil {
		return domain.Report{}, nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reports.read"); err != nil {
		return domain.Report{}, nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reports.download"); err != nil {
		return domain.Report{}, nil, err
	}

	report, err := s.repo.GetByID(ctx, tenantID, reportID)
	if err != nil {
		return domain.Report{}, nil, err
	}
	if report.StorageKey == nil || *report.StorageKey == "" {
		return domain.Report{}, nil, domain.ErrReportNotFound
	}

	reader, err := s.storage.Open(ctx, storage.GetObjectInput{
		ObjectKey: *report.StorageKey,
	})
	if err != nil {
		return domain.Report{}, nil, err
	}
	return report, reader, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, actorUserID, reportID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.reports"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reports.write"); err != nil {
		return err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reports.delete"); err != nil {
		return err
	}

	report, err := s.repo.GetByID(ctx, tenantID, reportID)
	if err != nil {
		return err
	}

	if report.StorageKey != nil && strings.TrimSpace(*report.StorageKey) != "" {
		if err := s.storage.Delete(ctx, storage.GetObjectInput{ObjectKey: *report.StorageKey}); err != nil {
			return err
		}
	}

	if err := s.repo.Delete(ctx, tenantID, reportID); err != nil {
		return err
	}

	_ = s.audit.Write(ctx, "finance.report.delete", "finance_report", reportID, report, map[string]any{"id": reportID, "deleted": true, "report_type": report.ReportType})
	return nil
}

func isSupportedFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv", "pdf":
		return true
	default:
		return false
	}
}

func normalizeReportType(reportType string) string {
	clean := strings.ToLower(strings.TrimSpace(reportType))
	if clean == "" {
		return "transactions"
	}
	return clean
}

func isSupportedReportType(reportType string) bool {
	switch reportType {
	case "transactions", "savings", "budgets", "reminders":
		return true
	default:
		return false
	}
}

func buildCSV(headers []string, records [][]string) io.Reader {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	_ = writer.Write(headers)
	for _, record := range records {
		_ = writer.Write(record)
	}
	writer.Flush()
	return buffer
}

func buildTransactionsCSV(rows []domain.TransactionRow) io.Reader {
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		account := row.AccountName
		if account == "" {
			account = row.AccountID
		}
		category := "-"
		if row.CategoryName != nil && strings.TrimSpace(*row.CategoryName) != "" {
			category = *row.CategoryName
		} else if row.CategoryID != nil {
			category = *row.CategoryID
		}
		records = append(records, []string{
			row.ID,
			row.InputDate.Format("2006-01-02"),
			row.TransactionDate.Format("2006-01-02"),
			row.Type,
			strconv.FormatInt(row.AmountMinor, 10),
			row.Currency,
			account,
			category,
			nullableString(row.Description),
			nullableString(row.MerchantName),
			nullableString(row.PaymentMethod),
		})
	}
	return buildCSV(
		[]string{"id", "input_date", "transaction_date", "type", "amount_minor", "currency", "account", "category", "description", "merchant", "payment_method"},
		records,
	)
}

func buildSavingsCSV(rows []domain.SavingsRow) io.Reader {
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		records = append(records, []string{
			row.ID,
			row.Name,
			strconv.FormatInt(row.TargetAmountMinor, 10),
			strconv.FormatInt(row.CurrentAmountMinor, 10),
			fmt.Sprintf("%.2f", row.ProgressPercent),
			row.Currency,
			formatDatePtr(row.StartDate),
			formatDatePtr(row.TargetDate),
			row.Status,
			row.UpdatedAt.Format(time.RFC3339),
		})
	}
	return buildCSV(
		[]string{"id", "name", "target_amount_minor", "current_amount_minor", "progress_percent", "currency", "start_date", "target_date", "status", "updated_at"},
		records,
	)
}

func buildBudgetsCSV(rows []domain.BudgetRow) io.Reader {
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		category := "-"
		if row.CategoryName != nil {
			category = *row.CategoryName
		} else if row.CategoryID != nil {
			category = *row.CategoryID
		}
		alert := ""
		if row.AlertThresholdPct != nil {
			alert = strconv.Itoa(*row.AlertThresholdPct)
		}
		records = append(records, []string{
			row.ID,
			row.Name,
			category,
			strconv.FormatInt(row.AmountLimitMinor, 10),
			row.Currency,
			row.Period,
			row.StartDate.Format("2006-01-02"),
			formatDatePtr(row.EndDate),
			alert,
			row.Status,
			row.UpdatedAt.Format(time.RFC3339),
		})
	}
	return buildCSV(
		[]string{"id", "name", "category", "amount_limit_minor", "currency", "period", "start_date", "end_date", "alert_threshold_pct", "status", "updated_at"},
		records,
	)
}

func buildRemindersCSV(rows []domain.ReminderRow) io.Reader {
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		amount := ""
		if row.AmountMinor != nil {
			amount = strconv.FormatInt(*row.AmountMinor, 10)
		}
		currency := ""
		if row.Currency != nil {
			currency = *row.Currency
		}
		records = append(records, []string{
			row.ID,
			row.Title,
			nullableString(row.Description),
			amount,
			currency,
			row.DueDate.Format("2006-01-02"),
			row.RepeatInterval,
			row.Status,
			formatDateTimePtr(row.LastTriggeredAt),
			row.UpdatedAt.Format(time.RFC3339),
		})
	}
	return buildCSV(
		[]string{"id", "title", "description", "amount_minor", "currency", "due_date", "repeat_interval", "status", "last_triggered_at", "updated_at"},
		records,
	)
}

func renderTransactionsPDF(rows []domain.TransactionRow) ([]byte, error) {
	headers := []string{"ID", "Date", "Type", "Amount", "Currency", "Account", "Category", "Merchant"}
	// Re-tuned for more columns
	widths := []float64{18, 22, 16, 32, 16, 52, 71, 50}
	records := make([][]string, 0, len(rows))

	categoryTotals := make(map[string]int64)
	var totalExpense int64
	currency := "IDR"

	for _, row := range rows {
		account := row.AccountName
		if account == "" {
			account = shortID(row.AccountID)
		}
		category := "-"
		if row.CategoryName != nil && strings.TrimSpace(*row.CategoryName) != "" {
			category = *row.CategoryName
		} else if row.CategoryID != nil {
			category = shortID(*row.CategoryID)
		}

		if row.Type == "expense" {
			categoryTotals[category] += row.AmountMinor
			totalExpense += row.AmountMinor
			currency = row.Currency
		}

		records = append(records, []string{
			shortID(row.ID),
			row.TransactionDate.Format("2006-01-02"),
			row.Type,
			formatMoney(row.Currency, row.AmountMinor),
			row.Currency,
			account,
			category,
			nullableString(row.MerchantName),
		})
	}

	// Prepare Summary Records
	summaryHeaders := []string{"Category Summary", "Total Amount"}
	summaryWidths := []float64{138, 139}
	summaryRecords := make([][]string, 0)
	for cat, total := range categoryTotals {
		summaryRecords = append(summaryRecords, []string{cat, formatMoney(currency, total)})
	}
	summaryRecords = append(summaryRecords, []string{"GRAND TOTAL EXPENSE", formatMoney(currency, totalExpense)})

	return renderTablePDFWithSummary("Pekan - Transactions Report", headers, widths, records, summaryHeaders, summaryWidths, summaryRecords)
}

func renderSavingsPDF(rows []domain.SavingsRow) ([]byte, error) {
	headers := []string{"ID", "Name", "Target", "Current", "Progress", "Currency", "Start", "Target Date", "Status"}
	widths := []float64{22, 26, 22, 22, 18, 16, 18, 20, 22}
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		records = append(records, []string{
			shortID(row.ID),
			row.Name,
			formatMoney(row.Currency, row.TargetAmountMinor),
			formatMoney(row.Currency, row.CurrentAmountMinor),
			fmt.Sprintf("%.2f%%", row.ProgressPercent),
			row.Currency,
			formatDatePtr(row.StartDate),
			formatDatePtr(row.TargetDate),
			row.Status,
		})
	}
	return renderTablePDF("Pekan - Savings Report", headers, widths, records)
}

func renderBudgetsPDF(rows []domain.BudgetRow) ([]byte, error) {
	headers := []string{"ID", "Name", "Category", "Limit", "Period", "Start", "End", "Status"}
	widths := []float64{22, 28, 26, 24, 20, 20, 20, 26}
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		category := "-"
		if row.CategoryName != nil {
			category = *row.CategoryName
		} else if row.CategoryID != nil {
			category = shortID(*row.CategoryID)
		}
		records = append(records, []string{
			shortID(row.ID),
			row.Name,
			category,
			formatMoney(row.Currency, row.AmountLimitMinor),
			row.Period,
			row.StartDate.Format("2006-01-02"),
			formatDatePtr(row.EndDate),
			row.Status,
		})
	}
	return renderTablePDF("Pekan - Budgets Report", headers, widths, records)
}

func renderRemindersPDF(rows []domain.ReminderRow) ([]byte, error) {
	headers := []string{"ID", "Title", "Due Date", "Amount", "Repeat", "Status", "Last Triggered"}
	widths := []float64{24, 40, 22, 26, 20, 20, 34}
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		amount := "-"
		if row.AmountMinor != nil {
			currency := "IDR"
			if row.Currency != nil {
				currency = *row.Currency
			}
			amount = formatMoney(currency, *row.AmountMinor)
		}
		records = append(records, []string{
			shortID(row.ID),
			row.Title,
			row.DueDate.Format("2006-01-02"),
			amount,
			row.RepeatInterval,
			row.Status,
			formatDateTimePtr(row.LastTriggeredAt),
		})
	}
	return renderTablePDF("Pekan - Reminders Report", headers, widths, records)
}

func renderTablePDFWithSummary(title string, headers []string, widths []float64, records [][]string, summaryHeaders []string, summaryWidths []float64, summaryRecords [][]string) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 12, 10)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 9, title)
	pdf.Ln(11)

	const (
		lineHeight = 4.6
		cellPad    = 1.1
	)

	renderHeader := func(h []string, w []float64) {
		pdf.SetFont("Helvetica", "B", 9)
		for i, head := range h {
			pdf.CellFormat(w[i], 7, head, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Helvetica", "", 7.5)
	}
	renderHeader(headers, widths)

	for _, row := range records {
		normalized := sanitizeRowValues(row, widths)
		rowHeight := calcRowHeight(pdf, normalized, widths, lineHeight, cellPad)
		if pdf.GetY()+rowHeight > 198 {
			pdf.AddPage()
			renderHeader(headers, widths)
		}
		drawRow(pdf, normalized, widths, lineHeight, cellPad, rowHeight)
	}

	if len(summaryRecords) > 0 {
		pdf.Ln(10)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.Cell(0, 9, "Category Summary")
		pdf.Ln(10)
		renderHeader(summaryHeaders, summaryWidths)
		for _, row := range summaryRecords {
			normalized := sanitizeRowValues(row, summaryWidths)
			rowHeight := calcRowHeight(pdf, normalized, summaryWidths, lineHeight, cellPad)
			if pdf.GetY()+rowHeight > 198 {
				pdf.AddPage()
				renderHeader(summaryHeaders, summaryWidths)
			}
			drawRow(pdf, normalized, summaryWidths, lineHeight, cellPad, rowHeight)
		}
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func renderTablePDF(title string, headers []string, widths []float64, records [][]string) ([]byte, error) {
	return renderTablePDFWithSummary(title, headers, widths, records, nil, nil, nil)
}

func calcRowHeight(pdf *gofpdf.Fpdf, values []string, widths []float64, lineHeight, padding float64) float64 {
	maxLines := 1
	for i, val := range values {
		lines := countWrappedLines(pdf, val, widths[i], padding)
		if lines > maxLines {
			maxLines = lines
		}
	}
	return float64(maxLines)*lineHeight + (padding * 2)
}

func countWrappedLines(pdf *gofpdf.Fpdf, value string, width, padding float64) int {
	contentWidth := width - (padding * 2)
	if contentWidth < 2 {
		contentWidth = 2
	}
	segments := strings.Split(value, "\n")
	totalLines := 0
	for _, segment := range segments {
		wrapped := pdf.SplitLines([]byte(segment), contentWidth)
		if len(wrapped) == 0 {
			totalLines++
			continue
		}
		totalLines += len(wrapped)
	}
	if totalLines == 0 {
		return 1
	}
	return totalLines
}

func drawRow(pdf *gofpdf.Fpdf, values []string, widths []float64, lineHeight, padding, rowHeight float64) {
	startX, startY := pdf.GetX(), pdf.GetY()
	for i, val := range values {
		cellX, cellY := pdf.GetX(), pdf.GetY()
		pdf.Rect(cellX, cellY, widths[i], rowHeight, "D")
		contentWidth := widths[i] - (padding * 2)
		if contentWidth < 2 {
			contentWidth = 2
		}
		pdf.SetXY(cellX+padding, cellY+padding)
		pdf.MultiCell(contentWidth, lineHeight, val, "", "L", false)
		pdf.SetXY(cellX+widths[i], cellY)
	}
	pdf.SetXY(startX, startY+rowHeight)
}

func sanitizeRowValues(values []string, widths []float64) []string {
	out := make([]string, len(values))
	for i, value := range values {
		width := 22.0
		if i < len(widths) {
			width = widths[i]
		}
		out[i] = sanitizeCellValue(value, width)
	}
	return out
}

func sanitizeCellValue(value string, width float64) string {
	clean := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n"))
	clean = strings.Join(strings.Fields(clean), " ")
	if clean == "" {
		return "-"
	}

	// Break long tokens (UUID/object keys) into smaller chunks so MultiCell can wrap.
	breakAt := int(width / 2)
	if breakAt < 12 {
		breakAt = 12
	}
	parts := strings.Split(clean, " ")
	for i, part := range parts {
		parts[i] = splitLongToken(part, breakAt)
	}
	out := strings.Join(parts, " ")

	runes := []rune(out)
	if len(runes) > 260 {
		return string(runes[:257]) + "..."
	}
	return out
}

func splitLongToken(token string, size int) string {
	runes := []rune(token)
	if len(runes) <= size {
		return token
	}
	// Keep money/number formats intact to avoid broken table rendering in PDF cells.
	if strings.ContainsAny(token, ".,") {
		return token
	}
	if isNumericToken(token) {
		return token
	}
	chunks := make([]string, 0, (len(runes)/size)+1)
	for idx := 0; idx < len(runes); idx += size {
		end := idx + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[idx:end]))
	}
	// Force explicit line-break opportunities for long IDs/object keys.
	return strings.Join(chunks, "\n")
}

func isNumericToken(token string) bool {
	for _, char := range token {
		if char < '0' || char > '9' {
			return false
		}
	}
	return len(token) > 0
}

func nullableString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func formatDatePtr(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.Format("2006-01-02")
}

func formatDateTimePtr(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.UTC().Format(time.RFC3339)
}

func formatMoney(currency string, amount int64) string {
	if strings.EqualFold(currency, "IDR") {
		return "Rp " + formatIDR(amount)
	}
	return fmt.Sprintf("%s %d", strings.ToUpper(currency), amount)
}

func formatIDR(amount int64) string {
	abs := amount
	if abs < 0 {
		abs = -abs
	}
	raw := strconv.FormatInt(abs, 10)
	n := len(raw)
	if n <= 3 {
		if amount < 0 {
			return "-" + raw
		}
		return raw
	}

	var b strings.Builder
	for i, ch := range raw {
		if i != 0 && (n-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	if amount < 0 {
		return "-" + b.String()
	}
	return b.String()
}

func shortID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	if len(trimmed) <= 8 {
		return strings.ToUpper(trimmed)
	}
	return strings.ToUpper(trimmed[:8])
}
