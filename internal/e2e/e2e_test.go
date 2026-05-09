package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"yunt/internal/api"
	"yunt/internal/api/handlers"
	"yunt/internal/api/middleware"
	"yunt/internal/config"
	"yunt/internal/repository/sqlite"
	"yunt/internal/service"
	smtpserver "yunt/internal/smtp"
)

type e2eEnv struct {
	SMTPAddr string
	APIAddr  string
	Repo     *sqlite.Repository
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func setupE2E(t *testing.T) *e2eEnv {
	t.Helper()

	poolCfg := &sqlite.ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}
	pool, err := sqlite.NewConnectionPool(poolCfg)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	repo, err := sqlite.NewWithOptions(pool, true, true)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create repo: %v", err)
	}

	authCfg := config.AuthConfig{
		JWTSecret:         "e2e-test-secret-key-32-chars-long!",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 1 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}

	sessionStore := service.NewInMemorySessionStore()
	authSvc := service.NewAuthService(authCfg, repo.Users(), sessionStore)
	userSvc := service.NewUserService(authCfg, repo.Users())
	mailboxSvc := service.NewMailboxService(repo, nil)
	messageSvc := service.NewMessageService(repo, nil)
	webhookSvc := service.NewWebhookService(repo, nil)

	_ = userSvc

	// SMTP server
	smtpPort := freePort(t)
	smtpCfg := smtpserver.NewDefaultConfig()
	smtpCfg.Host = "127.0.0.1"
	smtpCfg.Port = smtpPort
	smtpCfg.MaxMessageSize = 10 * 1024 * 1024

	logger := config.NewDefaultLogger()
	smtpSrv, err := smtpserver.New(smtpCfg, logger.Logger,
		smtpserver.WithRepo(repo),
		smtpserver.WithMailboxRepo(repo.Mailboxes()),
		smtpserver.WithMessageRepo(repo.Messages()),
	)
	if err != nil {
		repo.Close()
		t.Fatalf("failed to create SMTP server: %v", err)
	}

	go smtpSrv.Start()
	time.Sleep(100 * time.Millisecond)

	// API server
	apiPort := freePort(t)
	apiCfg := config.APIConfig{
		Enabled:            true,
		Host:               "127.0.0.1",
		Port:               apiPort,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		CORSAllowedOrigins: []string{"*"},
	}

	apiSrv := api.New(apiCfg, api.WithLogger(logger))
	v1 := apiSrv.Router().V1()

	authHandler := handlers.NewAuthHandler(authSvc)
	authHandler.RegisterRoutes(v1)

	authMw := middleware.Auth(authSvc)
	authed := v1.Group("", authMw)

	msgHandler := handlers.NewMessageHandler(messageSvc, mailboxSvc, authSvc)
	msgHandler.RegisterRoutes(authed)

	mbHandler := handlers.NewMailboxHandler(mailboxSvc, authSvc)
	mbHandler.RegisterRoutes(authed)

	usersHandler := handlers.NewUsersHandler(userSvc, authSvc)
	usersHandler.RegisterRoutes(authed, authSvc)

	whHandler := handlers.NewWebhookHandler(webhookSvc, authSvc)
	whHandler.RegisterRoutes(authed)

	healthHandler := handlers.NewHealthHandler(repo, "e2e-test")
	healthHandler.RegisterRoutes(apiSrv.Echo())

	ctx, cancel := context.WithCancel(context.Background())
	go apiSrv.StartWithContext(ctx)
	time.Sleep(100 * time.Millisecond)

	smtpAddr := fmt.Sprintf("127.0.0.1:%d", smtpPort)
	apiAddr := fmt.Sprintf("http://127.0.0.1:%d", apiPort)

	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		smtpSrv.Stop(stopCtx)
		apiSrv.Shutdown(stopCtx)
		repo.Close()
	})

	return &e2eEnv{SMTPAddr: smtpAddr, APIAddr: apiAddr, Repo: repo}
}

func loginAdmin(t *testing.T, apiAddr string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin123"})
	resp, err := http.Post(apiAddr+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed with %d: %s", resp.StatusCode, string(b))
	}

	var raw map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&raw)
	data := raw["data"].(map[string]interface{})
	tokens := data["tokens"].(map[string]interface{})
	return tokens["accessToken"].(string)
}

func apiGet(t *testing.T, url, token string) (int, map[string]interface{}) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	var raw map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&raw)
	return resp.StatusCode, raw
}

func sendTestEmail(t *testing.T, smtpAddr, from, to, subject, body string) {
	t.Helper()
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nDate: %s\r\nMessage-ID: <%d@test.local>\r\n\r\n%s",
		from, to, subject, time.Now().Format(time.RFC1123Z), time.Now().UnixNano(), body)
	err := smtp.SendMail(smtpAddr, nil, from, []string{to}, []byte(msg))
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}
}

// --- Tests ---

func TestHealthEndpoints(t *testing.T) {
	env := setupE2E(t)

	tests := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/healthz", 200, "OK"},
		{"/ready", 200, "OK"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(env.APIAddr + tc.path)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, resp.StatusCode)
			}

			b, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(b), tc.wantBody) {
				t.Errorf("expected body to contain %q, got %q", tc.wantBody, string(b))
			}
		})
	}

	t.Run("/health", func(t *testing.T) {
		status, raw := apiGet(t, env.APIAddr+"/health", "")
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
		if data, ok := raw["data"].(map[string]interface{}); ok {
			if s, _ := data["status"].(string); s != "healthy" {
				t.Errorf("expected healthy, got %s", s)
			}
		}
	})
}

func TestAuthLogin(t *testing.T) {
	env := setupE2E(t)

	t.Run("successful login", func(t *testing.T) {
		token := loginAdmin(t, env.APIAddr)
		if token == "" {
			t.Error("expected non-empty token")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
		resp, err := http.Post(env.APIAddr+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestSendAndRetrieveMessage(t *testing.T) {
	env := setupE2E(t)

	sendTestEmail(t, env.SMTPAddr,
		"sender@example.com",
		"inbox@localhost",
		"E2E Test Subject",
		"This is the e2e test body.")

	time.Sleep(200 * time.Millisecond)

	token := loginAdmin(t, env.APIAddr)
	status, raw := apiGet(t, env.APIAddr+"/api/v1/messages", token)

	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}

	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data field, got %v", raw)
	}

	items, ok := data["items"].([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("expected at least 1 message, got %v", data)
	}

	msg := items[0].(map[string]interface{})

	if subj, _ := msg["subject"].(string); subj != "E2E Test Subject" {
		t.Errorf("expected subject 'E2E Test Subject', got %q", subj)
	}

	if from, ok := msg["from"].(map[string]interface{}); ok {
		if addr, _ := from["address"].(string); addr != "sender@example.com" {
			t.Errorf("expected from address 'sender@example.com', got %q", addr)
		}
	}
}

func TestSendMultipleMessages(t *testing.T) {
	env := setupE2E(t)

	for i := 0; i < 5; i++ {
		sendTestEmail(t, env.SMTPAddr,
			"sender@example.com",
			"inbox@localhost",
			fmt.Sprintf("Message %d", i+1),
			fmt.Sprintf("Body of message %d", i+1))
	}

	time.Sleep(300 * time.Millisecond)

	token := loginAdmin(t, env.APIAddr)
	status, raw := apiGet(t, env.APIAddr+"/api/v1/messages", token)

	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}

	data := raw["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) < 5 {
		t.Errorf("expected at least 5 messages, got %d", len(items))
	}
}

func TestMessageReadAndDelete(t *testing.T) {
	env := setupE2E(t)

	sendTestEmail(t, env.SMTPAddr,
		"sender@example.com",
		"inbox@localhost",
		"Delete Me",
		"This message will be deleted.")

	time.Sleep(200 * time.Millisecond)

	token := loginAdmin(t, env.APIAddr)
	_, raw := apiGet(t, env.APIAddr+"/api/v1/messages", token)
	data := raw["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) == 0 {
		t.Fatal("no messages found")
	}

	msgID := items[0].(map[string]interface{})["id"].(string)

	// Mark as read
	req, _ := http.NewRequest(http.MethodPost, env.APIAddr+"/api/v1/messages/"+msgID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("mark-as-read failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		t.Errorf("mark-as-read: expected 200/204, got %d", resp.StatusCode)
	}

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, env.APIAddr+"/api/v1/messages/"+msgID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		t.Errorf("delete: expected 200/204, got %d", resp.StatusCode)
	}

	// Verify deleted
	status, raw := apiGet(t, env.APIAddr+"/api/v1/messages/"+msgID, token)
	if status != 404 {
		t.Errorf("expected 404 after delete, got %d", status)
	}
}
