package storage

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type serviceAccountKey struct {
	Type        string `json:"type"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

type GoogleDriveStorage struct {
	folderID   string
	creds      serviceAccountKey
	privateKey *rsa.PrivateKey
	token      string
	tokenExp   time.Time
	mu         sync.Mutex
	httpClient *http.Client
}

func NewGoogleDriveStorage(folderID, credentialsJSON string) (*GoogleDriveStorage, error) {
	if strings.TrimSpace(credentialsJSON) == "" {
		return nil, errors.New("google drive credentials not provided")
	}

	var key serviceAccountKey
	if err := json.Unmarshal([]byte(credentialsJSON), &key); err != nil {
		return nil, fmt.Errorf("invalid google drive credentials json: %w", err)
	}

	if key.ClientEmail == "" || key.PrivateKey == "" {
		return nil, errors.New("invalid service account key: missing client_email or private_key")
	}

	parsedKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(key.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	if key.TokenURI == "" {
		key.TokenURI = "https://oauth2.googleapis.com/token"
	}

	return &GoogleDriveStorage{
		folderID:   folderID,
		creds:      key,
		privateKey: parsedKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (s *GoogleDriveStorage) getAccessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Before(s.tokenExp.Add(-2*time.Minute)) {
		return s.token, nil
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iss":   s.creds.ClientEmail,
		"scope": "https://www.googleapis.com/auth/drive",
		"aud":   s.creds.TokenURI,
		"exp":   now.Add(1 * time.Hour).Unix(),
		"iat":   now.Unix(),
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedJWT, err := tokenObj.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt assertion: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", signedJWT)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.creds.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("empty access token received from google oauth")
	}

	s.token = tokenResp.AccessToken
	if tokenResp.ExpiresIn <= 0 {
		tokenResp.ExpiresIn = 3600
	}
	s.tokenExp = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return s.token, nil
}

func (s *GoogleDriveStorage) Put(ctx context.Context, in PutObjectInput) (PutObjectOutput, error) {
	token, err := s.getAccessToken(ctx)
	if err != nil {
		return PutObjectOutput{}, fmt.Errorf("gdrive auth failed: %w", err)
	}

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	// Part 1: Metadata
	metaHeader := make(textproto.MIMEHeader)
	metaHeader.Set("Content-Type", "application/json; charset=UTF-8")
	metaPart, err := writer.CreatePart(metaHeader)
	if err != nil {
		return PutObjectOutput{}, err
	}

	metadata := map[string]any{
		"name": in.ObjectKey,
	}
	if s.folderID != "" {
		metadata["parents"] = []string{s.folderID}
	}
	metaBytes, _ := json.Marshal(metadata)
	if _, err := metaPart.Write(metaBytes); err != nil {
		return PutObjectOutput{}, err
	}

	// Part 2: Media Body
	mediaHeader := make(textproto.MIMEHeader)
	cType := in.ContentType
	if cType == "" {
		cType = "application/octet-stream"
	}
	mediaHeader.Set("Content-Type", cType)
	mediaPart, err := writer.CreatePart(mediaHeader)
	if err != nil {
		return PutObjectOutput{}, err
	}

	if _, err := io.Copy(mediaPart, in.Body); err != nil {
		return PutObjectOutput{}, fmt.Errorf("failed to write media body: %w", err)
	}

	if err := writer.Close(); err != nil {
		return PutObjectOutput{}, err
	}

	uploadURL := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bodyBuf)
	if err != nil {
		return PutObjectOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "multipart/related; boundary="+writer.Boundary())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return PutObjectOutput{}, fmt.Errorf("gdrive upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		return PutObjectOutput{}, fmt.Errorf("gdrive upload returned status %d: %s", resp.StatusCode, string(errBody))
	}

	return PutObjectOutput{
		Provider:  "gdrive",
		ObjectKey: in.ObjectKey,
	}, nil
}

func (s *GoogleDriveStorage) findFileID(ctx context.Context, token, objectKey string) (string, error) {
	escapedName := strings.ReplaceAll(objectKey, "'", "\\'")
	query := fmt.Sprintf("name = '%s' and trashed = false", escapedName)
	if s.folderID != "" {
		query = fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", escapedName, s.folderID)
	}

	searchURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s&fields=files(id,name)&pageSize=1", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gdrive file lookup failed (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResult struct {
		Files []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return "", err
	}

	if len(searchResult.Files) == 0 {
		return "", errors.New("file not found on google drive")
	}

	return searchResult.Files[0].ID, nil
}

func (s *GoogleDriveStorage) Open(ctx context.Context, in GetObjectInput) (io.ReadCloser, error) {
	token, err := s.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("gdrive auth failed: %w", err)
	}

	fileID, err := s.findFileID(ctx, token, in.ObjectKey)
	if err != nil {
		return nil, err
	}

	downloadURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gdrive download failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gdrive download returned status %d: %s", resp.StatusCode, string(errBody))
	}

	return resp.Body, nil
}

func (s *GoogleDriveStorage) Delete(ctx context.Context, in GetObjectInput) error {
	token, err := s.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("gdrive auth failed: %w", err)
	}

	fileID, err := s.findFileID(ctx, token, in.ObjectKey)
	if err != nil {
		// If file doesn't exist, deletion is considered a no-op
		return nil
	}

	deleteURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gdrive delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gdrive delete returned status %d: %s", resp.StatusCode, string(errBody))
	}

	return nil
}
