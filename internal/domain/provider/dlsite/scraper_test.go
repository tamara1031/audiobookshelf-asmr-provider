package dlsite

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestDLsiteFetcher_Search_NonRJQuery(t *testing.T) {
	p := NewDLsiteFetcher()
	results, err := p.Search(context.Background(), "some keyword")
	if err != nil {
		t.Fatalf("expected no error for keyword search, got: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for keyword search, got %v", results)
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
