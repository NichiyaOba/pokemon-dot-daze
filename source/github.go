// Package source obtains the contribution count that drives the artwork.
package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultAPIURL is GitHub's GraphQL endpoint.
const DefaultAPIURL = "https://api.github.com/graphql"

// maxResponseBytes caps how much of a response body is read, so a malformed
// or hostile endpoint cannot exhaust memory.
const maxResponseBytes = 4 << 20

// GitHub reads a user's lifetime contribution total from the GraphQL API.
//
// The API caps contributionsCollection at one year per range, so the total is
// assembled by querying each year since the account was created and summing.
type GitHub struct {
	// Login is the GitHub username to read contributions for.
	Login string
	// Token authenticates the API call. A token is always required; the
	// contributions API rejects anonymous requests.
	Token string
	// Client defaults to a client with a 30s timeout when nil.
	Client *http.Client
	// APIURL defaults to DefaultAPIURL when empty.
	APIURL string
	// Now defaults to time.Now when nil. Injectable for tests.
	Now func() time.Time
}

// TotalContributions returns the user's contribution count across their whole
// account history.
func (g GitHub) TotalContributions(ctx context.Context) (int, error) {
	if strings.TrimSpace(g.Login) == "" {
		return 0, fmt.Errorf("a GitHub login is required")
	}
	if strings.TrimSpace(g.Token) == "" {
		return 0, fmt.Errorf("a GitHub token is required: set GITHUB_TOKEN or pass --token")
	}

	createdAt, err := g.accountCreatedAt(ctx)
	if err != nil {
		return 0, fmt.Errorf("looking up when %s joined GitHub: %w", g.Login, err)
	}

	// createdAt and the from/to ranges are UTC, so the current year must be
	// taken in UTC as well. Using local time drops a whole year for anyone
	// west of UTC during the first hours of 1 January.
	total, err := g.sumYears(ctx, createdAt.UTC().Year(), g.now().UTC().Year())
	if err != nil {
		return 0, fmt.Errorf("summing contributions for %s: %w", g.Login, err)
	}
	return total, nil
}

func (g GitHub) accountCreatedAt(ctx context.Context) (time.Time, error) {
	const query = `query($login:String!){user(login:$login){createdAt}}`

	var resp struct {
		User struct {
			CreatedAt time.Time `json:"createdAt"`
		} `json:"user"`
	}
	if err := g.do(ctx, query, map[string]any{"login": g.Login}, &resp); err != nil {
		return time.Time{}, err
	}
	if resp.User.CreatedAt.IsZero() {
		return time.Time{}, fmt.Errorf("the API returned no creation date for user %q", g.Login)
	}
	return resp.User.CreatedAt, nil
}

// sumYears fetches every year in one request using GraphQL aliases, so the
// number of round trips stays at two regardless of account age.
func (g GitHub) sumYears(ctx context.Context, firstYear, lastYear int) (int, error) {
	if firstYear > lastYear {
		return 0, fmt.Errorf("account creation year %d is after the current year %d", firstYear, lastYear)
	}

	query, aliases := buildYearlyQuery(firstYear, lastYear)

	var resp struct {
		User map[string]json.RawMessage `json:"user"`
	}
	if err := g.do(ctx, query, map[string]any{"login": g.Login}, &resp); err != nil {
		return 0, err
	}

	total := 0
	for _, alias := range aliases {
		raw, ok := resp.User[alias]
		if !ok {
			return 0, fmt.Errorf("the API response is missing the %q field", alias)
		}
		var year struct {
			ContributionCalendar struct {
				TotalContributions int `json:"totalContributions"`
			} `json:"contributionCalendar"`
		}
		if err := json.Unmarshal(raw, &year); err != nil {
			return 0, fmt.Errorf("decoding %q: %w", alias, err)
		}
		total += year.ContributionCalendar.TotalContributions
	}
	return total, nil
}

// buildYearlyQuery returns the aliased query and the alias names in order.
func buildYearlyQuery(firstYear, lastYear int) (string, []string) {
	var b strings.Builder
	aliases := make([]string, 0, lastYear-firstYear+1)

	b.WriteString("query($login:String!){user(login:$login){")
	for y := firstYear; y <= lastYear; y++ {
		alias := fmt.Sprintf("y%d", y)
		aliases = append(aliases, alias)
		fmt.Fprintf(&b, `%s:contributionsCollection(from:"%d-01-01T00:00:00Z",to:"%d-12-31T23:59:59Z"){contributionCalendar{totalContributions}}`, alias, y, y)
	}
	b.WriteString("}}")

	return b.String(), aliases
}

// graphQLError is one entry of a GraphQL errors array.
type graphQLError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// do sends a GraphQL request and decodes the data field into out.
func (g GitHub) do(ctx context.Context, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return fmt.Errorf("encoding the request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.apiURL(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pokemon-dot-daze")

	resp, err := g.client().Do(req)
	if err != nil {
		return fmt.Errorf("calling the GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Read one byte past the cap so truncation is detectable rather than
	// surfacing later as a confusing JSON decode error.
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("reading the response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return fmt.Errorf("the GitHub API response exceeded %d bytes", maxResponseBytes)
	}

	if err := checkHTTPStatus(resp, payload); err != nil {
		return err
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decoding the response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("the GitHub API reported an error: %s", joinGraphQLErrors(envelope.Errors))
	}
	if len(envelope.Data) == 0 {
		return fmt.Errorf("the GitHub API returned no data")
	}

	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decoding the response data: %w", err)
	}
	return nil
}

func checkHTTPStatus(resp *http.Response, payload []byte) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	snippet := strings.TrimSpace(string(payload))
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("the GitHub token was rejected (401): check it is set and not expired")
	case http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return fmt.Errorf("the GitHub API rate limit is exhausted (resets at %s)", resp.Header.Get("X-RateLimit-Reset"))
		}
		return fmt.Errorf("the GitHub API refused the request (403): %s", snippet)
	default:
		return fmt.Errorf("the GitHub API returned HTTP %d: %s", resp.StatusCode, snippet)
	}
}

func joinGraphQLErrors(errs []graphQLError) string {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		if e.Type != "" {
			msgs = append(msgs, fmt.Sprintf("%s: %s", e.Type, e.Message))
			continue
		}
		msgs = append(msgs, e.Message)
	}
	return strings.Join(msgs, "; ")
}

func (g GitHub) apiURL() string {
	if g.APIURL != "" {
		return g.APIURL
	}
	return DefaultAPIURL
}

func (g GitHub) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (g GitHub) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}
