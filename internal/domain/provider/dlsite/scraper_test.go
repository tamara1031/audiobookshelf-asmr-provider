package dlsite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestFetcher creates a dlsiteFetcher pointing at a test server URL.
func newTestFetcher(baseURL string) *dlsiteFetcher {
	f := NewDLsiteFetcher().(*dlsiteFetcher)
	f.baseURL = baseURL
	return f
}

func TestDLsiteFetcher_Search_RJCode(t *testing.T) {
	mockHTML := `
    <html>
        <body>
            <h1 id="work_name">Test Work Title</h1>
            <span class="maker_name"><a href="#">Test Circle</a></span>
            <div class="product_slider_data">
                <div data-src="//example.com/cover.jpg"></div>
            </div>
            <table id="work_outline">
                <tr><th>販売日</th><td><a href="#">2023年01月01日</a></td></tr>
                <tr><th>ジャンル</th><td><a href="#">Tag1</a><a href="#">Tag2</a></td></tr>
                <tr><th>声優</th><td><a href="#">Actor1</a></td></tr>
            </table>
        </body>
    </html>
    `

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/maniax/work/=/product_id/RJ010101.html" {
			t.Errorf("Expected path /maniax/work/=/product_id/RJ010101.html, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := f.Search(ctx, "RJ010101")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Title != "Test Work Title" {
		t.Errorf("Expected title 'Test Work Title', got '%s'", result.Title)
	}
	if result.Author != "Test Circle" {
		t.Errorf("Expected author 'Test Circle', got '%s'", result.Author)
	}
	if result.PublishedYear != "2023-01-01" {
		t.Errorf("Expected date '2023-01-01', got '%s'", result.PublishedYear)
	}
	if result.Cover != "https://example.com/cover.jpg" {
		t.Errorf("Expected cover 'https://example.com/cover.jpg', got '%s'", result.Cover)
	}
}

func TestDLsiteFetcher_Search_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "RJ999999")
	if err == nil {
		t.Error("Expected error for 404, got nil")
	}
	if results != nil {
		t.Error("Expected nil results for 404")
	}
}

func TestDLsiteFetcher_ID(t *testing.T) {
	p := NewDLsiteFetcher()
	if p.ID() != "dlsite" {
		t.Errorf("expected ID 'dlsite', got %q", p.ID())
	}
}

func TestDLsiteFetcher_CacheTTL(t *testing.T) {
	p := NewDLsiteFetcher()
	if p.CacheTTL() != 24*time.Hour {
		t.Errorf("expected CacheTTL 24h, got %v", p.CacheTTL())
	}
}

func TestDLsiteFetcher_Search_NetworkError(t *testing.T) {
	f := newTestFetcher("http://127.0.0.1:1") // unreachable port

	_, err := f.Search(context.Background(), "RJ123456")
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

func TestDLsiteFetcher_ExtractCoverURL_SrcFallback(t *testing.T) {
	mockHTML := `
	<html><body>
		<div class="product_slider_data">
			<div src="//example.com/fallback.jpg"></div>
		</div>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "RJ010101")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Cover != "https://example.com/fallback.jpg" {
		t.Errorf("expected cover with https prefix, got %q", results[0].Cover)
	}
}

func TestDLsiteFetcher_ExtractCoverURL_AbsoluteURL(t *testing.T) {
	mockHTML := `
	<html><body>
		<div class="product_slider_data">
			<div data-src="https://example.com/absolute.jpg"></div>
		</div>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "RJ010101")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Cover != "https://example.com/absolute.jpg" {
		t.Errorf("expected absolute URL unchanged, got %q", results[0].Cover)
	}
}

func TestDLsiteFetcher_ExtractCoverURL_NoCover(t *testing.T) {
	mockHTML := `<html><body><h1 id="work_name">No Cover</h1></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "RJ010101")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Cover != "" {
		t.Errorf("expected empty cover, got %q", results[0].Cover)
	}
}

func TestDLsiteFetcher_Search_Keyword(t *testing.T) {
	mockHTML := `
	<html><body>
		<table id="search_result_list">
			<tr>
				<td class="work_name"><a href="https://www.dlsite.com/maniax/work/=/product_id/RJ999999.html">Keyword Match 1</a></td>
				<td class="maker_name"><a href="#">Maker 1</a></td>
			</tr>
			<tr>
				<td class="work_name"><a href="https://www.dlsite.com/maniax/work/=/product_id/RJ888888.html">Keyword Match 2</a></td>
				<td class="maker_name"><a href="#">Maker 2</a></td>
			</tr>
		</table>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/fsr/") {
			t.Errorf("Expected fsr search path, got %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "/keyword/") {
			t.Errorf("Expected keyword search path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "some keyword")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Keyword Match 1" {
		t.Errorf("Expected title 'Keyword Match 1', got '%s'", results[0].Title)
	}
	if results[0].ISBN != "RJ999999" {
		t.Errorf("Expected ISBN 'RJ999999', got '%s'", results[0].ISBN)
	}
}

func TestDLsiteFetcher_Search_KeywordWithSpaces(t *testing.T) {
	mockHTML := `<html><body><table id="search_result_list"></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that spaces are encoded as '+' (which is what QueryEscape does)
		// Go's http server might decode the path before we see it in r.URL.Path?
		// r.URL.Path usually has decoded path. r.URL.RawPath has encoded.
		// However, QueryEscape puts '+' in the path. standard path escaping uses %20.
		// If we use QueryEscape for path segment, it might be interpreted literally.

		// Let's check the RawPath or RequestURI to be sure how it was sent.
		if !strings.Contains(r.RequestURI, "foo+bar") {
			t.Errorf("Expected URL to contain 'foo+bar', got %s", r.RequestURI)
		}
		if strings.Contains(r.RequestURI, "foo%20bar") {
			t.Errorf("URL should not contain 'foo%%20bar'")
		}
		if !strings.Contains(r.RequestURI, "/fsr/") {
			t.Errorf("Expected fsr search path, got %s", r.RequestURI)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	_, err := f.Search(context.Background(), "foo bar")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
}
