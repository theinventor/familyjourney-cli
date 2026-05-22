package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theinventor/familyjourney-cli/internal/client"
	"github.com/theinventor/familyjourney-cli/internal/config"
	"github.com/theinventor/familyjourney-cli/internal/credstore"
	"github.com/theinventor/familyjourney-cli/internal/exitcode"
)

type captured struct {
	method      string
	path        string
	rawQuery    string
	authHeader  string
	contentType string
	userAgent   string
	body        []byte
	hits        int
}

func runHTTPCommand(t *testing.T, argv []string, status int, body string) (string, string, *captured, error) {
	t.Helper()
	cap := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.hits++
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.rawQuery = r.URL.RawQuery
		cap.authHeader = r.Header.Get("Authorization")
		cap.contentType = r.Header.Get("Content-Type")
		cap.userAgent = r.Header.Get("User-Agent")
		cap.body, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv(client.EnvAPIURL, srv.URL)
	t.Setenv(client.EnvAPIToken, "fj_token_1234567890")
	t.Setenv(config.EnvConfig, t.TempDir()+"/config.json")
	t.Setenv(credstore.EnvDisableKeychain, "1")

	stdout, stderr, err := runCommand(t, argv)
	if err != nil {
		stderr += "familyjourney: " + err.Error() + "\n"
	}
	return stdout, stderr, cap, err
}

func runCommand(t *testing.T, argv []string) (string, string, error) {
	t.Helper()
	root := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(argv)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func decodeBody(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal request body: %v; raw=%s", err, raw)
	}
	return got
}

func TestWhoamiRequestShape(t *testing.T) {
	stdout, _, cap, err := runHTTPCommand(t, []string{"whoami"}, http.StatusOK, `{"user":{"email":"parent@example.com"}}`)
	if err != nil {
		t.Fatalf("whoami returned error: %v", err)
	}
	if cap.hits != 1 || cap.method != http.MethodGet || cap.path != "/api/v1/auth/me" {
		t.Fatalf("expected GET /api/v1/auth/me once, got hits=%d %s %s", cap.hits, cap.method, cap.path)
	}
	if cap.authHeader != "Bearer fj_token_1234567890" {
		t.Fatalf("auth header mismatch: %q", cap.authHeader)
	}
	if !strings.Contains(stdout, "parent@example.com") {
		t.Fatalf("stdout should pass through JSON body, got %q", stdout)
	}
}

func TestKidCreateRequestShape(t *testing.T) {
	_, _, cap, err := runHTTPCommand(t, []string{"kids", "create", "--name", "Avery", "--email", "avery@example.com", "--group-id", "7,8"}, http.StatusCreated, `{"id":1}`)
	if err != nil {
		t.Fatalf("kids create returned error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/api/v1/kids" {
		t.Fatalf("expected POST /api/v1/kids, got %s %s", cap.method, cap.path)
	}
	body := decodeBody(t, cap.body)
	kid := body["kid"].(map[string]any)
	if kid["name"] != "Avery" || kid["email"] != "avery@example.com" {
		t.Fatalf("unexpected kid body: %#v", kid)
	}
	groups := kid["group_ids"].([]any)
	if fmt.Sprint(groups) != "[7 8]" {
		t.Fatalf("unexpected group ids: %#v", groups)
	}
}

func TestBadgePublishRequestShape(t *testing.T) {
	_, _, cap, err := runHTTPCommand(t, []string{"badges", "publish", "42"}, http.StatusOK, `{"id":42,"status":"published"}`)
	if err != nil {
		t.Fatalf("badges publish returned error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/api/v1/badges/42/publish" {
		t.Fatalf("expected publish path, got %s %s", cap.method, cap.path)
	}
	if len(cap.body) != 0 {
		t.Fatalf("publish should not send a body, got %s", cap.body)
	}
}

func TestSubmissionDenyRequestShape(t *testing.T) {
	_, _, cap, err := runHTTPCommand(t, []string{"submissions", "deny", "55", "--reason", "Needs a clearer photo"}, http.StatusOK, `{"id":55,"status":"denied"}`)
	if err != nil {
		t.Fatalf("submissions deny returned error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/api/v1/badge_submissions/55/deny" {
		t.Fatalf("expected deny path, got %s %s", cap.method, cap.path)
	}
	body := decodeBody(t, cap.body)
	if body["reason"] != "Needs a clearer photo" {
		t.Fatalf("unexpected deny body: %#v", body)
	}
}

func TestPrizeCreateRequestShape(t *testing.T) {
	_, _, cap, err := runHTTPCommand(t, []string{"prizes", "create", "--name", "Movie night", "--description", "Pick the movie", "--point-cost", "50", "--active=false"}, http.StatusCreated, `{"id":3}`)
	if err != nil {
		t.Fatalf("prizes create returned error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/api/v1/prizes" {
		t.Fatalf("expected POST /api/v1/prizes, got %s %s", cap.method, cap.path)
	}
	body := decodeBody(t, cap.body)
	prize := body["prize"].(map[string]any)
	if prize["name"] != "Movie night" || prize["point_cost"].(float64) != 50 || prize["active"].(bool) {
		t.Fatalf("unexpected prize body: %#v", prize)
	}
}

func TestRedemptionDenyRequestShape(t *testing.T) {
	_, _, cap, err := runHTTPCommand(t, []string{"redemptions", "deny", "77", "--feedback", "Not this week"}, http.StatusOK, `{"id":77,"status":"denied"}`)
	if err != nil {
		t.Fatalf("redemptions deny returned error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/api/v1/redemptions/77/deny" {
		t.Fatalf("expected redemption deny path, got %s %s", cap.method, cap.path)
	}
	body := decodeBody(t, cap.body)
	if body["feedback"] != "Not this week" {
		t.Fatalf("unexpected redemption deny body: %#v", body)
	}
}

func TestChallengeCreateRequestShape(t *testing.T) {
	_, _, cap, err := runHTTPCommand(t, []string{"challenges", "create", "--badge-id", "12", "--description", "Practice daily", "--position", "2"}, http.StatusCreated, `{"id":9}`)
	if err != nil {
		t.Fatalf("challenges create returned error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/api/v1/challenges" {
		t.Fatalf("expected POST /api/v1/challenges, got %s %s", cap.method, cap.path)
	}
	body := decodeBody(t, cap.body)
	if body["badge_id"].(float64) != 12 {
		t.Fatalf("unexpected badge_id body: %#v", body)
	}
	challenge := body["challenge"].(map[string]any)
	if challenge["description"] != "Practice daily" || challenge["position"].(float64) != 2 {
		t.Fatalf("unexpected challenge body: %#v", challenge)
	}
}

func TestNon2xxReturnsWrappedExitCode(t *testing.T) {
	stdout, stderr, _, err := runHTTPCommand(t, []string{"kids", "get", "404"}, http.StatusNotFound, `{"error":"Kid not found"}`)
	if err == nil {
		t.Fatalf("expected non-2xx to return an error")
	}
	if exitcode.ExitCodeFor(err) != exitcode.NotFound {
		t.Fatalf("expected not-found exit code, got %d", exitcode.ExitCodeFor(err))
	}
	if !strings.Contains(stdout, "Kid not found") {
		t.Fatalf("server error body should be printed to stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "HTTP 404") {
		t.Fatalf("stderr should include status after main-style rendering, got %q", stderr)
	}
}

func TestDeleteRequiresForceBeforeRequest(t *testing.T) {
	_, stderr, cap, err := runHTTPCommand(t, []string{"kids", "delete", "1"}, http.StatusOK, `{"ok":true}`)
	if err == nil {
		t.Fatalf("delete without --force should fail")
	}
	if cap.hits != 0 {
		t.Fatalf("delete without --force should not hit API, got %d hits", cap.hits)
	}
	if exitcode.ExitCodeFor(err) != exitcode.Usage {
		t.Fatalf("expected usage exit code, got %d; stderr=%q", exitcode.ExitCodeFor(err), stderr)
	}
}

func TestAuthSaveMasksAndPersistsFileProfile(t *testing.T) {
	t.Setenv(config.EnvConfig, t.TempDir()+"/config.json")
	t.Setenv(credstore.EnvDisableKeychain, "1")
	stdout, _, err := runCommand(t, []string{"auth", "save", "--profile", "parent", "--api-token", "fj_secret_1234567890", "--api-url", "https://example.test", "--storage", "file"})
	if err != nil {
		t.Fatalf("auth save returned error: %v", err)
	}
	if strings.Contains(stdout, "fj_secret_1234567890") {
		t.Fatalf("auth save must not print full token: %q", stdout)
	}
	if !strings.Contains(stdout, "fj_secre...7890") {
		t.Fatalf("auth save should print a masked fingerprint, got %q", stdout)
	}
	f, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	p, ok := f.Get("parent")
	if !ok || p.APIToken != "fj_secret_1234567890" || p.APIURL != "https://example.test" {
		t.Fatalf("saved profile mismatch: ok=%v profile=%#v", ok, p)
	}
}

func TestSkillGetOutputsCLIFirstSkill(t *testing.T) {
	stdout, _, err := runCommand(t, []string{"skill", "get", "familyjourney"})
	if err != nil {
		t.Fatalf("skill get returned error: %v", err)
	}
	for _, want := range []string{"familyjourney whoami", "familyjourney submissions approve", "parent-only API"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("skill output missing %q", want)
		}
	}
	if strings.Contains(stdout, "fj.py") {
		t.Fatalf("skill should not direct agents to the old Python helper")
	}
}
