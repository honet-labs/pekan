package usecase

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"pekan/backend/internal/modules/finance/receipts/domain"
	"pekan/backend/internal/platform/imageutil"
	"pekan/backend/internal/platform/storage"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo       domain.Repository
	authz      Authorizer
	audit      AuditLogger
	secretKey  string
	legacyKeys []string
	client     *http.Client
	storage    storage.ObjectStorage
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger, secretKey string, legacyKeys []string, storageProvider storage.ObjectStorage) *Service {
	seen := map[string]struct{}{}
	filtered := make([]string, 0, len(legacyKeys)+1)
	for _, key := range append([]string{secretKey}, legacyKeys...) {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		filtered = append(filtered, trimmed)
	}
	return &Service{repo: repo, authz: authz, audit: audit, secretKey: secretKey, legacyKeys: filtered, client: &http.Client{Timeout: 75 * time.Second}, storage: storageProvider}
}

type UpsertProviderInput struct {
	TenantID     string
	ActorUserID  string
	ProviderCode string
	DisplayName  string
	BaseURL      *string
	ModelName    string
	IsEnabled    bool
	APIKey       string
	ClearAPIKey  bool
}

type ScanReceiptInput struct {
	TenantID         string
	ActorUserID      string
	ProviderCode     string
	OriginalFilename string
	MimeType         string
	Content          []byte
}

type ConfigStatusOutput struct {
	HasConfiguredProvider bool
	ActiveProviders       []string
}

type ModelOption struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type TestProviderConnectionInput struct {
	TenantID     string
	ProviderCode string
	BaseURL      string
	APIKey       string
	ModelName    string
}

type TestProviderConnectionOutput struct {
	ProviderCode     string
	Models           []ModelOption
	UsingSavedAPIKey bool
	BaseURL          string
}

type ScanReceiptOutput struct {
	Scan       domain.ReceiptScan
	Extraction domain.ReceiptExtraction
	Draft      DraftTransactionSuggestion
}

type DraftTransactionSuggestion struct {
	Type                 string                 `json:"type"`
	CategoryName         string                 `json:"category_name,omitempty"`
	AmountMinor          int64                  `json:"amount_minor"`
	Currency             string                 `json:"currency"`
	Date                 string                 `json:"transaction_date,omitempty"`
	Description          string                 `json:"description,omitempty"`
	MerchantName         string                 `json:"merchant_name,omitempty"`
	ReceiptNumber        string                 `json:"receipt_number,omitempty"`
	PaymentMethod        string                 `json:"payment_method,omitempty"`
	SubtotalMinor        int64                  `json:"subtotal_minor,omitempty"`
	TaxMinor             int64                  `json:"tax_minor,omitempty"`
	ServiceChargeMinor   int64                  `json:"service_charge_minor,omitempty"`
	ReceiptDiscountMinor int64                  `json:"receipt_discount_minor,omitempty"`
	Confidence           float64                `json:"confidence,omitempty"`
	Items                []DraftTransactionItem `json:"items,omitempty"`
}

type DraftTransactionItem struct {
	ItemName          string  `json:"item_name"`
	Quantity          float64 `json:"quantity"`
	PriceMinor        int64   `json:"price_per_unit_minor"`
	DiscountMinor     int64   `json:"discount_minor,omitempty"`
	TotalMinor        int64   `json:"total_minor"`
	Notes             string  `json:"notes,omitempty"`
}

var supportedProviders = map[string]struct {
	Name         string
	DefaultURL   string
	DefaultModel string
}{
	"openai":            {Name: "OpenAI", DefaultURL: "https://api.openai.com/v1", DefaultModel: "gpt-4o-mini"},
	"gemini":            {Name: "Google Gemini", DefaultURL: "https://generativelanguage.googleapis.com", DefaultModel: "gemini-2.0-flash"},
	"anthropic":         {Name: "Anthropic Claude", DefaultURL: "https://api.anthropic.com/v1", DefaultModel: "claude-3-5-sonnet-latest"},
	"sumopod":           {Name: "SumoPod AI", DefaultURL: "https://ai.sumopod.com/v1", DefaultModel: "gpt-4o-mini"},
	"openai_compatible": {Name: "OpenAI Compatible", DefaultURL: "https://api.sumopod.com/v1", DefaultModel: "gpt-4o-mini"},
}

func (s *Service) ListProviderConfigs(ctx context.Context, tenantID string) ([]domain.ProviderConfig, error) {
	if err := s.ensureSettingsAccess(ctx); err != nil {
		return nil, err
	}
	items, err := s.repo.ListProviderConfigs(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byCode := map[string]domain.ProviderConfig{}
	for _, item := range items {
		byCode[item.ProviderCode] = sanitizeConfig(item)
	}
	out := make([]domain.ProviderConfig, 0, len(supportedProviders))
	for code, meta := range supportedProviders {
		if item, ok := byCode[code]; ok {
			out = append(out, item)
			continue
		}
		baseURL := meta.DefaultURL
		out = append(out, domain.ProviderConfig{ProviderCode: code, DisplayName: meta.Name, BaseURL: &baseURL, ModelName: meta.DefaultModel, IsEnabled: false, HasAPIKey: false})
	}
	return out, nil
}

func sanitizeConfig(item domain.ProviderConfig) domain.ProviderConfig {
	item.APIKeyCiphertext = nil
	item.ModelName = normalizeProviderModel(item.ProviderCode, item.ModelName)
	if strings.TrimSpace(item.ModelName) == "" {
		item.ModelName = supportedProviders[item.ProviderCode].DefaultModel
	}
	return item
}

func (s *Service) UpsertProviderConfigs(ctx context.Context, inputs []UpsertProviderInput) ([]domain.ProviderConfig, error) {
	if err := s.ensureSettingsAccess(ctx); err != nil {
		return nil, err
	}
	if len(inputs) == 0 {
		return []domain.ProviderConfig{}, nil
	}
	out := make([]domain.ProviderConfig, 0, len(inputs))
	now := time.Now().UTC()
	for _, in := range inputs {
		code := normalizeProviderCode(in.ProviderCode)
		meta, ok := supportedProviders[code]
		if !ok {
			return nil, domain.ErrInvalidProvider
		}
		var encrypted *string
		if strings.TrimSpace(in.APIKey) != "" {
			cipherText, err := encryptSecret(s.secretKey, strings.TrimSpace(in.APIKey))
			if err != nil {
				return nil, err
			}
			encrypted = &cipherText
		}
		if in.ClearAPIKey {
			empty := ""
			encrypted = &empty
		}
		displayName := strings.TrimSpace(in.DisplayName)
		if displayName == "" {
			displayName = meta.Name
		}
		modelName := normalizeProviderModel(code, in.ModelName)
		if modelName == "" {
			modelName = meta.DefaultModel
		}
		var baseURL *string
		if in.BaseURL != nil {
			trimmed := strings.TrimSpace(*in.BaseURL)
			if trimmed != "" {
				baseURL = &trimmed
			}
		}
		if baseURL == nil && meta.DefaultURL != "" {
			baseURL = &meta.DefaultURL
		}
		saved, err := s.repo.UpsertProviderConfig(ctx, domain.ProviderConfig{
			TenantID: in.TenantID, ProviderCode: code, DisplayName: displayName, BaseURL: baseURL, ModelName: modelName,
			IsEnabled: in.IsEnabled, APIKeyCiphertext: encrypted, CreatedBy: in.ActorUserID, UpdatedBy: in.ActorUserID,
			CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, sanitizeConfig(saved))
	}
	_ = s.audit.Write(ctx, "finance.receipt_scan.settings.update", "finance_receipt_provider_configs", inputs[0].TenantID, nil, map[string]any{"count": len(out)})
	return out, nil
}

func (s *Service) GetConfigStatus(ctx context.Context, tenantID string) (ConfigStatusOutput, error) {
	if err := s.ensureScanAccess(ctx); err != nil {
		return ConfigStatusOutput{}, err
	}
	items, err := s.repo.ListProviderConfigs(ctx, tenantID)
	if err != nil {
		return ConfigStatusOutput{}, err
	}
	out := ConfigStatusOutput{}
	for _, item := range items {
		if item.IsEnabled && item.HasAPIKey {
			out.HasConfiguredProvider = true
			out.ActiveProviders = append(out.ActiveProviders, item.ProviderCode)
		}
	}

	// 3. Also check global settings for supported providers
	activeGlobal, _, _ := s.repo.GetGlobalSetting(ctx, "receipt_active_ai_provider")
	activeGlobal = normalizeProviderCode(activeGlobal)

	for code := range supportedProviders {
		globalKey := fmt.Sprintf("receipt_api_key_%s", code)
		val, _, err := s.repo.GetGlobalSetting(ctx, globalKey)
		if err == nil && val != "" {
			out.HasConfiguredProvider = true
			
			// If this is the global active provider, or no active provider is set yet, add it
			isGlobalActive := (activeGlobal == "" && code == "gemini") || (activeGlobal == code)
			
			found := false
			for _, p := range out.ActiveProviders {
				if p == code {
					found = true
					break
				}
			}
			if !found && (isGlobalActive || len(out.ActiveProviders) == 0) {
				out.ActiveProviders = append(out.ActiveProviders, code)
			}
		}
	}

	return out, nil
}

func (s *Service) TestProviderConnection(ctx context.Context, in TestProviderConnectionInput) (TestProviderConnectionOutput, error) {
	if err := s.ensureSettingsAccess(ctx); err != nil {
		return TestProviderConnectionOutput{}, err
	}
	code := normalizeProviderCode(in.ProviderCode)
	meta, ok := supportedProviders[code]
	if !ok {
		return TestProviderConnectionOutput{}, domain.ErrInvalidProvider
	}
	baseURL := strings.TrimSpace(in.BaseURL)
	if baseURL == "" {
		baseURL = meta.DefaultURL
	}
	apiKey := strings.TrimSpace(in.APIKey)
	usingSavedAPIKey := false
	if apiKey == "" {
		item, err := s.repo.GetProviderConfig(ctx, in.TenantID, code)
		if err != nil || !item.HasAPIKey {
			return TestProviderConnectionOutput{}, domain.ErrProviderNotConfigured
		}
		var decErr error
		apiKey, decErr = s.decryptSavedAPIKey(deref(item.APIKeyCiphertext))
		if decErr != nil {
			return TestProviderConnectionOutput{}, fmt.Errorf("%w: please save the API key again", domain.ErrProviderCredentialInvalid)
		}
		usingSavedAPIKey = true
		if strings.TrimSpace(baseURL) == "" && item.BaseURL != nil {
			baseURL = strings.TrimSpace(*item.BaseURL)
		}
	}
	models, err := s.listProviderModels(ctx, code, baseURL, apiKey)
	if err != nil {
		return TestProviderConnectionOutput{}, err
	}
	selectedModel := normalizeProviderModel(code, in.ModelName)
	if selectedModel == "" {
		selectedModel = meta.DefaultModel
	}
	models = mergeModelOption(models, ModelOption{ID: selectedModel, Label: selectedModel})
	return TestProviderConnectionOutput{
		ProviderCode:     code,
		Models:           models,
		UsingSavedAPIKey: usingSavedAPIKey,
		BaseURL:          baseURL,
	}, nil
}

func (s *Service) ListReceiptScans(ctx context.Context, tenantID string, limit int) ([]domain.ReceiptScan, error) {
	if err := s.ensureScanAccess(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListReceiptScans(ctx, tenantID, limit)
}

func (s *Service) DeleteReceiptScan(ctx context.Context, tenantID, scanID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return err
	}
	return s.repo.DeleteReceiptScan(ctx, tenantID, scanID)
}

func (s *Service) ClearReceiptScans(ctx context.Context, tenantID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return err
	}
	return s.repo.ClearReceiptScans(ctx, tenantID)
}

func (s *Service) GetReceiptScanImage(ctx context.Context, tenantID, scanID string) ([]byte, string, error) {
	if err := s.ensureScanAccess(ctx); err != nil {
		return nil, "", err
	}
	// Fetch the scan to get mime type and verify tenant
	scans, err := s.repo.ListReceiptScans(ctx, tenantID, 100)
	if err != nil {
		return nil, "", err
	}
	
	var scan domain.ReceiptScan
	found := false
	for _, s := range scans {
		if s.ID == scanID {
			scan = s
			found = true
			break
		}
	}
	
	if !found {
		return nil, "", domain.ErrInvalidFile
	}
	
	fileKey := fmt.Sprintf("%s/finance/receipt-scan/%s", tenantID, scanID)
	rc, err := s.storage.Open(ctx, storage.GetObjectInput{ObjectKey: fileKey})
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}
	return data, scan.MimeType, nil
}

func (s *Service) ScanReceipt(ctx context.Context, in ScanReceiptInput) (ScanReceiptOutput, error) {
	if err := s.ensureScanAccess(ctx); err != nil {
		return ScanReceiptOutput{}, err
	}
	if len(in.Content) == 0 {
		return ScanReceiptOutput{}, domain.ErrInvalidFile
	}
	mimeType := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !isSupportedMimeType(mimeType) {
		return ScanReceiptOutput{}, domain.ErrInvalidFile
	}

	activeProviders := s.resolveAllActiveProviders(ctx, in.TenantID, in.ProviderCode)
	if len(activeProviders) == 0 {
		return ScanReceiptOutput{}, domain.ErrNoConfiguredProvider
	}

	primary := activeProviders[0]
	scan, err := s.repo.CreateReceiptScan(ctx, domain.ReceiptScan{
		TenantID: in.TenantID, ProviderCode: primary.config.ProviderCode, ModelName: primary.config.ModelName, Status: "processing",
		OriginalFilename: in.OriginalFilename, MimeType: mimeType, CreatedBy: in.ActorUserID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		return ScanReceiptOutput{}, err
	}

	fileKey := fmt.Sprintf("%s/finance/receipt-scan/%s", in.TenantID, scan.ID)
	
	// Image Optimization: Compress JPEG and PNG using unified imageutil
	processedContent := in.Content
	if mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/jpg" {
		optimized, err := imageutil.CompressImage(bytes.NewReader(in.Content), mimeType)
		if err == nil {
			processedContent = optimized
		}
	}

	_, _ = s.storage.Put(ctx, storage.PutObjectInput{
		TenantID:    in.TenantID,
		Module:      "finance.receipts",
		ObjectKey:   fileKey,
		ContentType: mimeType,
		Body:        bytes.NewReader(processedContent),
	})

	var extraction domain.ReceiptExtraction
	var lastErr error
	var finalProvider domain.ProviderConfig
	success := false

	for _, p := range activeProviders {
		extraction, err = s.extractReceipt(ctx, p.config, p.apiKey, mimeType, processedContent)
		if err == nil {
			finalProvider = p.config
			success = true
			break
		}
		lastErr = err
	}

	if !success {
		msg := "All providers failed. Last error: " + lastErr.Error()
		scan.Status = "failed"
		scan.ErrorMessage = &msg
		scan.UpdatedAt = time.Now().UTC()
		_, _ = s.repo.UpdateReceiptScanResult(ctx, scan)
		return ScanReceiptOutput{}, err
	}
	payload, _ := json.Marshal(extraction)
	scan.Status = "completed"
	scan.ProviderCode = finalProvider.ProviderCode
	scan.ModelName = finalProvider.ModelName
	scan.ExtractedJSON = payload
	scan.ErrorMessage = nil
	scan.UpdatedAt = time.Now().UTC()
	scan, err = s.repo.UpdateReceiptScanResult(ctx, scan)
	if err != nil {
		return ScanReceiptOutput{}, err
	}

	draft := buildDraftSuggestion(extraction)
	_ = s.audit.Write(ctx, "finance.receipt_scan.create", "finance_receipt_scan", scan.ID, nil, map[string]any{"provider": finalProvider.ProviderCode})
	return ScanReceiptOutput{Scan: scan, Extraction: extraction, Draft: draft}, nil
}

func (s *Service) ensureSettingsAccess(ctx context.Context) error {
	if err := s.authz.EnsureModule(ctx, "finance.settings"); err != nil {
		return err
	}
	return s.authz.EnsurePermission(ctx, "finance.settings.read")
}

func (s *Service) ensureScanAccess(ctx context.Context) error {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return err
	}
	return s.authz.EnsurePermission(ctx, "finance.transactions.create")
}

func normalizeProviderCode(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func normalizeProviderModel(providerCode, model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return ""
	}
	switch normalizeProviderCode(providerCode) {
	case "gemini":
		value := strings.ToLower(trimmed)
		value = strings.TrimPrefix(value, "models/")
		value = strings.ReplaceAll(value, "_", "-")
		value = strings.Join(strings.Fields(value), "-")
		value = regexp.MustCompile(`-+`).ReplaceAllString(value, "-")
		value = regexp.MustCompile(`^gemini-([0-9]+)-([0-9]+)(.*)$`).ReplaceAllString(value, "gemini-$1.$2$3")
		return value
	default:
		return trimmed
	}
}

func isSupportedMimeType(v string) bool {
	return v == "image/jpeg" || v == "image/jpg" || v == "image/png" || v == "image/webp"
}

func (s *Service) resolveProviderConfig(ctx context.Context, tenantID, requested string) (domain.ProviderConfig, string, error) {
	code := normalizeProviderCode(requested)
	
	// 1. If not requested, check global active provider
	if code == "" {
		active, _, err := s.repo.GetGlobalSetting(ctx, "receipt_active_ai_provider")
		if err == nil && active != "" {
			code = normalizeProviderCode(active)
		}
	}
	
	if code == "" {
		code = "gemini" // Ultimate fallback
	}

	// 2. Try tenant config
	item, err := s.repo.GetProviderConfig(ctx, tenantID, code)
	if err == nil && item.IsEnabled && item.HasAPIKey {
		apiKey, err := s.decryptSavedAPIKey(deref(item.APIKeyCiphertext))
		if err == nil {
			item.ModelName = normalizeProviderModel(item.ProviderCode, item.ModelName)
			return item, apiKey, nil
		}
	}

	// 3. Fallback to Global Settings from Admin
	globalKey := fmt.Sprintf("receipt_api_key_%s", code)
	globalCipherText, isEnc, err := s.repo.GetGlobalSetting(ctx, globalKey)
	if err == nil && globalCipherText != "" {
		apiKey := globalCipherText
		if isEnc {
			apiKey, err = s.decryptSavedAPIKey(globalCipherText)
		}
		
		if err == nil && apiKey != "" {
			meta := supportedProviders[code]
			baseURL := meta.DefaultURL
			
			// Try to get custom model from global settings
			modelKey := fmt.Sprintf("receipt_model_%s", code)
			customModel, _, _ := s.repo.GetGlobalSetting(ctx, modelKey)
			if customModel == "" {
				customModel = meta.DefaultModel
			}

			return domain.ProviderConfig{
				ProviderCode: code,
				DisplayName:  meta.Name,
				BaseURL:      &baseURL,
				ModelName:    customModel,
				IsEnabled:    true,
				HasAPIKey:    true,
			}, apiKey, nil
		}
	}

	return domain.ProviderConfig{}, "", domain.ErrNoConfiguredProvider
}

func (s *Service) decryptSavedAPIKey(cipherText string) (string, error) {
	keys := s.legacyKeys
	if len(keys) == 0 {
		keys = []string{s.secretKey}
	}
	var lastErr error
	for _, key := range keys {
		plain, err := decryptSecret(key, cipherText)
		if err == nil {
			return plain, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("unable to decrypt saved API key")
	}
	return "", lastErr
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func encryptSecret(secret, plain string) (string, error) {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptSecret(secret, cipherText string) (string, error) {
	if strings.TrimSpace(cipherText) == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid cipher text")
	}
	nonce, enc := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func buildDraftSuggestion(ex domain.ReceiptExtraction) DraftTransactionSuggestion {
	currency := strings.ToUpper(strings.TrimSpace(ex.Currency))
	if currency == "" {
		currency = "IDR"
	}
	typ := strings.ToLower(strings.TrimSpace(ex.SuggestedType))
	if typ == "" {
		typ = "expense"
	}
	items := make([]DraftTransactionItem, 0, len(ex.Items))
	for _, item := range ex.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		qty := item.Qty
		if qty <= 0 {
			qty = 1
		}
		unitMinor := convertToMinor(currency, item.UnitPrice)
		discountMinor := convertToMinor(currency, item.Discount)
		totalMinor := convertToMinor(currency, item.LineTotal)
		if totalMinor <= 0 {
			totalMinor = int64(math.Round(qty*float64(unitMinor))) - discountMinor
		}
		if totalMinor <= 0 {
			totalMinor = int64(math.Round(qty * float64(unitMinor)))
		}
		notes := strings.TrimSpace(item.CategoryHint)
		items = append(items, DraftTransactionItem{
			ItemName:          name,
			Quantity:          qty,
			PriceMinor:        unitMinor,
			DiscountMinor:     discountMinor,
			TotalMinor:        totalMinor,
			Notes:             notes,
		})
	}
	subtotalMinor := convertToMinor(currency, ex.Subtotal)
	if subtotalMinor <= 0 && len(items) > 0 {
		for _, item := range items {
			subtotalMinor += item.TotalMinor
		}
	}
	taxMinor := convertToMinor(currency, ex.Tax)
	serviceMinor := convertToMinor(currency, ex.ServiceCharge)
	receiptDiscountMinor := convertToMinor(currency, ex.Discount)
	amountMinor := convertToMinor(currency, ex.Total)
	if amountMinor <= 0 {
		if convertToMinor(currency, ex.Subtotal) > 0 {
			amountMinor = subtotalMinor + serviceMinor - receiptDiscountMinor
		} else {
			amountMinor = subtotalMinor + taxMinor + serviceMinor - receiptDiscountMinor
		}
		if amountMinor < 0 {
			amountMinor = 0
		}
	}
	return DraftTransactionSuggestion{
		Type:                 typ,
		CategoryName:         strings.TrimSpace(ex.SuggestedCategoryName),
		AmountMinor:          amountMinor,
		Currency:             currency,
		Date:                 ex.TransactionDate,
		Description:          buildDescription(ex),
		MerchantName:         strings.TrimSpace(ex.MerchantName),
		ReceiptNumber:        strings.TrimSpace(ex.ReceiptNumber),
		PaymentMethod:        strings.TrimSpace(ex.PaymentMethod),
		SubtotalMinor:        subtotalMinor,
		TaxMinor:             taxMinor,
		ServiceChargeMinor:   serviceMinor,
		ReceiptDiscountMinor: receiptDiscountMinor,
		Confidence:           ex.Confidence,
		Items:                items,
	}
}

func buildDescription(ex domain.ReceiptExtraction) string {
	if strings.TrimSpace(ex.Description) != "" {
		return strings.TrimSpace(ex.Description)
	}
	parts := []string{}
	if ex.MerchantName != "" {
		parts = append(parts, ex.MerchantName)
	}
	if len(ex.Items) > 0 {
		itemNames := make([]string, 0, min(2, len(ex.Items)))
		for i, item := range ex.Items {
			if i >= 2 {
				break
			}
			if strings.TrimSpace(item.Name) != "" {
				itemNames = append(itemNames, strings.TrimSpace(item.Name))
			}
		}
		if len(itemNames) > 0 {
			parts = append(parts, strings.Join(itemNames, ", "))
		}
	}
	return strings.Join(parts, " - ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func convertToMinor(currency string, amount float64) int64 {
	multiplier := 100.0
	switch strings.ToUpper(currency) {
	case "IDR", "JPY", "KRW":
		multiplier = 1
	}
	return int64(math.Round(amount * multiplier))
}

func receiptPrompt() string {
	return `You are PEKAN AI Assistant, an expert in Indonesian receipt analysis.
Analyze the receipt image carefully. Follow these steps:
1. Identify the merchant type (Retail, Restaurant, Parking, etc.)
2. Look for the absolute final payment amount.
3. Extract items, quantities, and prices.

Return JSON only with this schema:
{
  "merchant_name": "string",
  "receipt_number": "string",
  "transaction_date": "YYYY-MM-DD",
  "currency": "IDR",
  "subtotal": 0,
  "total": 0,
  "tax": 0,
  "service_charge": 0,
  "discount": 0,
  "payment_method": "cash|debit|credit|ewallet|unknown",
  "suggested_type": "expense|income|transfer|savings",
  "suggested_category_name": "string",
  "description": "string",
  "confidence": 0.0,
  "notes": "Contextual analysis here",
  "items": [{"name":"string","qty":1,"unit_price":0,"discount":0,"line_total":0,"category_hint":"string"}]
}

IMPORTANT RULES:
- "total" MUST be the final amount actually paid (often labeled 'TOTAL BELANJA').
- TAX INCLUSIVE: In most Indonesian retail receipts (Indomaret, Alfamart), the prices already include tax. If 'TOTAL BELANJA' is shown, the 'total' field should be exactly that value. Do NOT add the 'tax' field to the 'total' field again.
- TAX EXCLUSIVE: If it's a restaurant bill, total usually = subtotal + tax + service - discount.
- For Indonesian receipts, "." is often a thousands separator and "," is a decimal separator. 
- Example: "25.000,00" should be extracted as 25000.
- Mathematical Consistency: unit_price * qty - item_discount = line_total.
- Do not include markdown fences. Return PURE JSON only.`
}

func (s *Service) listProviderModels(ctx context.Context, providerCode, baseURL, apiKey string) ([]ModelOption, error) {
	switch normalizeProviderCode(providerCode) {
	case "openai", "sumopod", "openai_compatible":
		return s.listOpenAICompatibleModels(ctx, baseURL, apiKey)
	case "gemini":
		return s.listGeminiModels(ctx, baseURL, apiKey)
	case "anthropic":
		return s.listAnthropicModels(ctx, baseURL, apiKey)
	default:
		return nil, domain.ErrInvalidProvider
	}
}

func (s *Service) listOpenAICompatibleModels(ctx context.Context, baseURL, apiKey string) ([]ModelOption, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/models"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider error: %s", compactProviderError(raw, resp.StatusCode))
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	items := make([]ModelOption, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		items = append(items, ModelOption{ID: id, Label: id})
	}
	return dedupeAndSortModelOptions(items), nil
}

func (s *Service) listGeminiModels(ctx context.Context, baseURL, apiKey string) ([]ModelOption, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/v1beta/models?key=" + url.QueryEscape(apiKey)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider error: %s", compactProviderError(raw, resp.StatusCode))
	}
	var parsed struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	items := make([]ModelOption, 0, len(parsed.Models))
	for _, item := range parsed.Models {
		if len(item.SupportedGenerationMethods) > 0 {
			supported := false
			for _, method := range item.SupportedGenerationMethods {
				if strings.EqualFold(strings.TrimSpace(method), "generateContent") {
					supported = true
					break
				}
			}
			if !supported {
				continue
			}
		}
		id := normalizeProviderModel("gemini", strings.TrimPrefix(strings.TrimSpace(item.Name), "models/"))
		if id == "" {
			continue
		}
		label := strings.TrimSpace(item.DisplayName)
		if label == "" {
			label = id
		}
		items = append(items, ModelOption{ID: id, Label: label})
	}
	return dedupeAndSortModelOptions(items), nil
}

func (s *Service) listAnthropicModels(ctx context.Context, baseURL, apiKey string) ([]ModelOption, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/models"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider error: %s", compactProviderError(raw, resp.StatusCode))
	}
	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
		Models []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	items := make([]ModelOption, 0, len(parsed.Data)+len(parsed.Models))
	appendItems := func(source []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}) {
		for _, item := range source {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			label := strings.TrimSpace(item.DisplayName)
			if label == "" {
				label = id
			}
			items = append(items, ModelOption{ID: id, Label: label})
		}
	}
	appendItems(parsed.Data)
	appendItems(parsed.Models)
	return dedupeAndSortModelOptions(items), nil
}

func dedupeAndSortModelOptions(items []ModelOption) []ModelOption {
	if len(items) == 0 {
		return []ModelOption{}
	}
	seen := map[string]ModelOption{}
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = id
		}
		seen[id] = ModelOption{ID: id, Label: label}
	}
	out := make([]ModelOption, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label)
	})
	return out
}

func mergeModelOption(items []ModelOption, item ModelOption) []ModelOption {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return dedupeAndSortModelOptions(items)
	}
	items = append(items, ModelOption{ID: id, Label: strings.TrimSpace(item.Label)})
	return dedupeAndSortModelOptions(items)
}

func (s *Service) extractReceipt(ctx context.Context, provider domain.ProviderConfig, apiKey, mimeType string, content []byte) (domain.ReceiptExtraction, error) {
	code := provider.ProviderCode
	switch code {
	case "openai", "sumopod", "openai_compatible":
		return s.extractOpenAICompatible(ctx, provider, apiKey, mimeType, content)
	case "gemini":
		return s.extractGemini(ctx, provider, apiKey, mimeType, content)
	case "anthropic":
		return s.extractAnthropic(ctx, provider, apiKey, mimeType, content)
	default:
		return domain.ReceiptExtraction{}, domain.ErrInvalidProvider
	}
}

func (s *Service) extractOpenAICompatible(ctx context.Context, provider domain.ProviderConfig, apiKey, mimeType string, content []byte) (domain.ReceiptExtraction, error) {
	baseURL := strings.TrimRight(deref(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = supportedProviders[provider.ProviderCode].DefaultURL
	}
	payload := map[string]any{
		"model":       provider.ModelName,
		"temperature": 0,
		"messages": []any{
			map[string]any{"role": "system", "content": receiptPrompt()},
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "text", "text": "Read this shopping receipt image and extract the fields."},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURL(mimeType, content)}},
			}},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.ReceiptExtraction{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return domain.ReceiptExtraction{}, fmt.Errorf("provider error: %s", compactProviderError(raw, resp.StatusCode))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.ReceiptExtraction{}, err
	}
	if len(parsed.Choices) == 0 {
		return domain.ReceiptExtraction{}, errors.New("provider returned no choices")
	}
	return parseExtractionJSON(parsed.Choices[0].Message.Content)
}

func (s *Service) extractGemini(ctx context.Context, provider domain.ProviderConfig, apiKey, mimeType string, content []byte) (domain.ReceiptExtraction, error) {
	baseURL := strings.TrimRight(deref(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = supportedProviders[provider.ProviderCode].DefaultURL
	}
	model := normalizeProviderModel(provider.ProviderCode, provider.ModelName)
	if strings.TrimSpace(model) == "" {
		model = supportedProviders[provider.ProviderCode].DefaultModel
	}
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, model, apiKey)
	payload := map[string]any{
		"contents": []any{map[string]any{"parts": []any{
			map[string]any{"text": receiptPrompt()},
			map[string]any{"inline_data": map[string]any{"mime_type": mimeType, "data": base64.StdEncoding.EncodeToString(content)}},
		}}},
		"generationConfig": map[string]any{"temperature": 0, "responseMimeType": "application/json"},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.ReceiptExtraction{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return domain.ReceiptExtraction{}, fmt.Errorf("provider error: %s", compactProviderError(raw, resp.StatusCode))
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.ReceiptExtraction{}, err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return domain.ReceiptExtraction{}, errors.New("provider returned no candidates")
	}
	return parseExtractionJSON(parsed.Candidates[0].Content.Parts[0].Text)
}

func (s *Service) extractAnthropic(ctx context.Context, provider domain.ProviderConfig, apiKey, mimeType string, content []byte) (domain.ReceiptExtraction, error) {
	baseURL := strings.TrimRight(deref(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = supportedProviders[provider.ProviderCode].DefaultURL
	}
	payload := map[string]any{
		"model":      provider.ModelName,
		"max_tokens": 1200,
		"system":     receiptPrompt(),
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": "Read this shopping receipt image and extract the fields."},
			map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": mimeType, "data": base64.StdEncoding.EncodeToString(content)}},
		}}},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.ReceiptExtraction{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return domain.ReceiptExtraction{}, fmt.Errorf("provider error: %s", compactProviderError(raw, resp.StatusCode))
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.ReceiptExtraction{}, err
	}
	for _, item := range parsed.Content {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			return parseExtractionJSON(item.Text)
		}
	}
	return domain.ReceiptExtraction{}, errors.New("provider returned no text")
}

func dataURL(mimeType string, content []byte) string {
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content))
}

var fencedJSONRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\}|\\[.*?\\])\\s*```")

func parseExtractionJSON(raw string) (domain.ReceiptExtraction, error) {
	jsonText := strings.TrimSpace(raw)
	if matches := fencedJSONRegex.FindStringSubmatch(jsonText); len(matches) > 1 {
		jsonText = strings.TrimSpace(matches[1])
	}
	if !strings.HasPrefix(jsonText, "{") {
		start := strings.Index(jsonText, "{")
		end := strings.LastIndex(jsonText, "}")
		if start >= 0 && end > start {
			jsonText = jsonText[start : end+1]
		}
	}
	var out domain.ReceiptExtraction
	if err := json.Unmarshal([]byte(jsonText), &out); err != nil {
		return domain.ReceiptExtraction{}, err
	}
	if strings.TrimSpace(out.Description) == "" {
		out.Description = buildDescription(out)
	}
	return out, nil
}

func compactProviderError(raw []byte, status int) string {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) > 220 {
		trimmed = trimmed[:220] + "..."
	}
	if trimmed == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	return fmt.Sprintf("HTTP %d: %s", status, trimmed)
}


type providerWithKey struct {
	config domain.ProviderConfig
	apiKey string
}

func (s *Service) resolveAllActiveProviders(ctx context.Context, tenantID, requestedCode string) []providerWithKey {
	var out []providerWithKey
	seen := map[string]struct{}{}

	// 1. Priority: Requested Provider
	if requestedCode != "" {
		code := normalizeProviderCode(requestedCode)
		p, key, err := s.resolveProviderConfig(ctx, tenantID, code)
		if err == nil {
			out = append(out, providerWithKey{config: p, apiKey: key})
			seen[code] = struct{}{}
		}
	}

	// 2. Secondary: Tenant Enabled Providers (that were not requested)
	tenantConfigs, _ := s.repo.ListProviderConfigs(ctx, tenantID)
	for _, tc := range tenantConfigs {
		if !tc.IsEnabled || !tc.HasAPIKey {
			continue
		}
		code := normalizeProviderCode(tc.ProviderCode)
		if _, ok := seen[code]; ok {
			continue
		}
		key, err := s.decryptSavedAPIKey(deref(tc.APIKeyCiphertext))
		if err == nil {
			out = append(out, providerWithKey{config: sanitizeConfig(tc), apiKey: key})
			seen[code] = struct{}{}
		}
	}

	// 3. Tertiary: Global Providers (Fallback)
	activeGlobal, _, _ := s.repo.GetGlobalSetting(ctx, "receipt_active_ai_provider")
	activeGlobal = normalizeProviderCode(activeGlobal)
	
	// Try to add global providers in order (Gemini usually first if active)
	codes := []string{"gemini", "openai", "anthropic", "sumopod", "openai_compatible"}
	if activeGlobal != "" {
		// Move active global to front of candidates
		newCodes := []string{activeGlobal}
		for _, c := range codes {
			if c != activeGlobal {
				newCodes = append(newCodes, c)
			}
		}
		codes = newCodes
	}

	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		globalKey := fmt.Sprintf("receipt_api_key_%s", code)
		globalCipherText, isEnc, err := s.repo.GetGlobalSetting(ctx, globalKey)
		if err == nil && globalCipherText != "" {
			apiKey := globalCipherText
			if isEnc {
				apiKey, _ = s.decryptSavedAPIKey(globalCipherText)
			}
			if apiKey != "" {
				meta := supportedProviders[code]
				baseURL := meta.DefaultURL
				modelKey := fmt.Sprintf("receipt_model_%s", code)
				customModel, _, _ := s.repo.GetGlobalSetting(ctx, modelKey)
				if customModel == "" {
					customModel = meta.DefaultModel
				}
				out = append(out, providerWithKey{
					config: domain.ProviderConfig{
						ProviderCode: code, DisplayName: meta.Name, BaseURL: &baseURL, ModelName: customModel, IsEnabled: true, HasAPIKey: true,
					},
					apiKey: apiKey,
				})
				seen[code] = struct{}{}
			}
		}
	}

	return out
}
