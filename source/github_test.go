package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeAPI serves canned GraphQL responses and records the queries it received.
type fakeAPI struct {
	server  *httptest.Server
	queries []string
}

func newFakeAPI(t *testing.T, handler func(query string, w http.ResponseWriter)) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			return
		}
		var req struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decoding request: %v", err)
			return
		}
		f.queries = append(f.queries, req.Query)
		handler(req.Query, w)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func isCreatedAtQuery(q string) bool { return strings.Contains(q, "createdAt") }

// respondCreatedAndYears is the happy-path handler: a creation date, then a
// fixed contribution count for every requested year.
func respondCreatedAndYears(createdYear, perYear int) func(string, http.ResponseWriter) {
	return func(q string, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		if isCreatedAtQuery(q) {
			fmt.Fprintf(w, `{"data":{"user":{"createdAt":"%d-06-15T00:00:00Z"}}}`, createdYear)
			return
		}
		fields := map[string]any{}
		for _, alias := range aliasesIn(q) {
			fields[alias] = map[string]any{
				"contributionCalendar": map[string]any{"totalContributions": perYear},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"user": fields}})
	}
}

// aliasesIn extracts the yNNNN aliases from a query string.
func aliasesIn(q string) []string {
	var out []string
	for _, part := range strings.Split(q, "y") {
		if len(part) < 4 {
			continue
		}
		year := part[:4]
		if _, err := time.Parse("2006", year); err != nil {
			continue
		}
		if strings.HasPrefix(part[4:], ":contributionsCollection") {
			out = append(out, "y"+year)
		}
	}
	return out
}

func newSource(url string, now time.Time) GitHub {
	return GitHub{
		Login:  "octocat",
		Token:  "test-token",
		APIURL: url,
		Now:    func() time.Time { return now },
	}
}

func TestTotalContributionsSumsEveryYear(t *testing.T) {
	f := newFakeAPI(t, respondCreatedAndYears(2019, 100))
	g := newSource(f.server.URL, time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC))

	got, err := g.TotalContributions(context.Background())
	if err != nil {
		t.Fatalf("TotalContributions returned %v", err)
	}

	// 2019..2023 inclusive is 5 years at 100 each.
	if want := 500; got != want {
		t.Errorf("TotalContributions() = %d, want %d", got, want)
	}
	if len(f.queries) != 2 {
		t.Errorf("made %d requests, want 2 (createdAt, then all years)", len(f.queries))
	}
}

func TestTotalContributionsHandlesSingleYearAccount(t *testing.T) {
	f := newFakeAPI(t, respondCreatedAndYears(2024, 42))
	g := newSource(f.server.URL, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))

	got, err := g.TotalContributions(context.Background())
	if err != nil {
		t.Fatalf("TotalContributions returned %v", err)
	}
	if got != 42 {
		t.Errorf("TotalContributions() = %d, want 42", got)
	}
}

func TestBuildYearlyQueryCoversRangeInclusively(t *testing.T) {
	query, aliases := buildYearlyQuery(2020, 2023)

	want := []string{"y2020", "y2021", "y2022", "y2023"}
	if len(aliases) != len(want) {
		t.Fatalf("got %d aliases, want %d", len(aliases), len(want))
	}
	for i, a := range want {
		if aliases[i] != a {
			t.Errorf("alias %d = %q, want %q", i, aliases[i], a)
		}
		if !strings.Contains(query, a+":contributionsCollection") {
			t.Errorf("query does not select %q", a)
		}
	}
	// Each range must stay inside a single calendar year: the API rejects wider spans.
	if !strings.Contains(query, `from:"2020-01-01T00:00:00Z",to:"2020-12-31T23:59:59Z"`) {
		t.Error("query does not constrain a year to a single calendar year")
	}
}

func TestRequiresLoginAndToken(t *testing.T) {
	tests := []struct {
		name  string
		src   GitHub
		match string
	}{
		{"no login", GitHub{Token: "t"}, "login is required"},
		{"blank login", GitHub{Login: "   ", Token: "t"}, "login is required"},
		{"no token", GitHub{Login: "octocat"}, "token is required"},
		{"blank token", GitHub{Login: "octocat", Token: "  "}, "token is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.src.TotalContributions(context.Background())
			if err == nil {
				t.Fatal("TotalContributions() = nil error, want an error")
			}
			if !strings.Contains(err.Error(), tt.match) {
				t.Errorf("error %q does not mention %q", err, tt.match)
			}
		})
	}
}

func TestSendsBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, `{"data":{"user":{"createdAt":"2024-01-01T00:00:00Z"}}}`)
	}))
	defer srv.Close()

	g := newSource(srv.URL, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC))
	_, _ = g.TotalContributions(context.Background())

	if want := "Bearer test-token"; gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
}

func TestHTTPErrorsAreReported(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		headers map[string]string
		body    string
		match   string
	}{
		{"unauthorized", http.StatusUnauthorized, nil, `{"message":"Bad credentials"}`, "token was rejected"},
		{"rate limited", http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1700000000"}, ``, "rate limit is exhausted"},
		{"forbidden", http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "42"}, `nope`, "refused the request"},
		{"server error", http.StatusInternalServerError, nil, `boom`, "HTTP 500"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			g := newSource(srv.URL, time.Now())
			_, err := g.TotalContributions(context.Background())
			if err == nil {
				t.Fatal("TotalContributions() = nil error, want an error")
			}
			if !strings.Contains(err.Error(), tt.match) {
				t.Errorf("error %q does not mention %q", err, tt.match)
			}
		})
	}
}

func TestGraphQLErrorsAreReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"errors":[{"type":"NOT_FOUND","message":"Could not resolve to a User"}]}`)
	}))
	defer srv.Close()

	g := newSource(srv.URL, time.Now())
	_, err := g.TotalContributions(context.Background())
	if err == nil {
		t.Fatal("TotalContributions() = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "Could not resolve to a User") {
		t.Errorf("error %q does not carry the API message", err)
	}
	if !strings.Contains(err.Error(), "NOT_FOUND") {
		t.Errorf("error %q does not carry the API error type", err)
	}
}

func TestMissingCreatedAtIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"user":{}}}`)
	}))
	defer srv.Close()

	g := newSource(srv.URL, time.Now())
	if _, err := g.TotalContributions(context.Background()); err == nil {
		t.Error("TotalContributions() = nil error, want an error")
	}
}

func TestMalformedJSONIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json at all`)
	}))
	defer srv.Close()

	g := newSource(srv.URL, time.Now())
	if _, err := g.TotalContributions(context.Background()); err == nil {
		t.Error("TotalContributions() = nil error, want an error")
	}
}

func TestMissingYearFieldIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "createdAt") {
			fmt.Fprint(w, `{"data":{"user":{"createdAt":"2022-01-01T00:00:00Z"}}}`)
			return
		}
		// Year aliases requested, but the response omits them.
		fmt.Fprint(w, `{"data":{"user":{}}}`)
	}))
	defer srv.Close()

	g := newSource(srv.URL, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err := g.TotalContributions(context.Background())
	if err == nil {
		t.Fatal("TotalContributions() = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error %q does not explain the missing field", err)
	}
}

func TestContextCancellationIsPropagated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	g := newSource(srv.URL, time.Now())
	if _, err := g.TotalContributions(ctx); err == nil {
		t.Error("TotalContributions() = nil error, want a cancellation error")
	}
}

func TestDefaultsAreApplied(t *testing.T) {
	g := GitHub{}
	if g.apiURL() != DefaultAPIURL {
		t.Errorf("apiURL() = %q, want %q", g.apiURL(), DefaultAPIURL)
	}
	if g.client() == nil {
		t.Error("client() = nil, want a default client")
	}
	if g.now().IsZero() {
		t.Error("now() returned the zero time")
	}
}

func TestSumYearsRejectsInvertedRange(t *testing.T) {
	g := newSource("http://invalid.invalid", time.Now())
	if _, err := g.sumYears(context.Background(), 2025, 2020); err == nil {
		t.Error("sumYears with an inverted range = nil error, want an error")
	}
}
