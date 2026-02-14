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
	f.ageCheckDisabled = true // Default to true for existing tests
	return f
}

func TestDLsiteFetcher_AgeCheckCookie(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
		expected bool
	}{
		{"Disabled (Age check active)", false, false},
		{"Enabled (Age check bypassed)", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie("adult_checked")
				hasCookie := err == nil && cookie.Value == "1"
				if hasCookie != tt.expected {
					t.Errorf("expected adult_checked cookie: %v, got %v", tt.expected, hasCookie)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`<html><body></body></html>`))
			}))
			defer server.Close()

			f := NewDLsiteFetcher().(*dlsiteFetcher)
			f.baseURL = server.URL
			f.ageCheckDisabled = tt.disabled

			_, _ = f.fetchPage(context.Background(), server.URL)
		})
	}
}

func TestDLsiteFetcher_Search_RJCode(t *testing.T) {
	mockHTML := `
    <html>
        <body>
            <h1 id="work_name">Test Work Title</h1>
            <span class="maker_name"><a href="#">Test Circle</a></span>
            <div class="product-slider-data">
                <div data-src="//example.com/cover.jpg"></div>
            </div>
            <div class="work_parts_area">This is a<br>test description<br/>with breaks.</div>
            <table id="work_outline">
                <tr><th>販売日</th><td><a href="#">2023年01月01日</a></td></tr>
                <tr><th>ジャンル</th><td><a href="#">Tag1</a><a href="#">Tag2</a></td></tr>
                <tr><th>声優</th><td><a href="#">Actor1</a></td></tr>
                <tr><th>シリーズ名</th><td>Test Series</td></tr>
                <tr><th>シナリオ</th><td>Test Scenario</td></tr>
                <tr><th>作品形式</th><td>Test Format</td></tr>
                <tr><th>年齢指定</th><td>R-18</td></tr>
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
	if result.Author != "Test Scenario" { // author should prioritize scenario
		t.Errorf("Expected author 'Test Scenario', got '%s'", result.Author)
	}
	if result.PublishedYear != "2023-01-01" {
		t.Errorf("Expected date '2023-01-01', got '%s'", result.PublishedYear)
	}
	if result.Cover != "https://example.com/cover.jpg" {
		t.Errorf("Expected cover 'https://example.com/cover.jpg', got '%s'", result.Cover)
	}
	if result.Description != "This is a\ntest description\nwith breaks." {
		t.Errorf("Expected description 'This is a\\ntest description\\nwith breaks.', got '%q'", result.Description)
	}
	if result.Series != "Test Series" {
		t.Errorf("Expected series 'Test Series', got '%s'", result.Series)
	}
	if len(result.Genres) != 1 || result.Genres[0] != "Test Format" {
		t.Errorf("Expected genres ['Test Format'], got %v", result.Genres)
	}
	if len(result.Tags) != 2 || result.Tags[0] != "Tag1" || result.Tags[1] != "Tag2" {
		t.Errorf("Expected tags ['Tag1', 'Tag2'], got %v", result.Tags)
	}
	if !result.Explicit {
		t.Errorf("Expected explicit: true for R-18, got false")
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
		<div class="product-slider-data">
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
		<div class="product-slider-data">
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
		if strings.Contains(r.URL.Path, "/fsr/") && strings.Contains(r.URL.Path, "/keyword/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		// Allow product page requests (return 404 to fallback to partial metadata)
		if strings.Contains(r.URL.Path, "/product_id/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Fail other unexpected requests
		t.Errorf("Unexpected request path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
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

func TestDLsiteFetcher_Search_AuthorNarratorSplit(t *testing.T) {
	mockHTML := `
	<html><body>
		<table id="search_result_list">
			<tr>
				<td class="work_name"><a href="https://www.dlsite.com/maniax/work/=/product_id/RJ123456.html">Split Test</a></td>
				<td class="maker_name">
					<a href="#">Circle Name</a>
					<span class="separator">/</span>
					<span class="author"><a href="#">CV Name</a></span>
				</td>
				<td class="search_result_img_box_inner">
					<img src="//example.com/thumb.jpg" />
				</td>
			</tr>
		</table>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "fsr") || strings.Contains(r.URL.Path, "keyword") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "split test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.Author != "Circle Name" {
		t.Errorf("Expected Author 'Circle Name', got '%s'", res.Author)
	}
	if res.Narrator != "CV Name" {
		t.Errorf("Expected Narrator 'CV Name', got '%s'", res.Narrator)
	}
	if res.Cover != "https://example.com/thumb.jpg" {
		t.Errorf("Expected Cover 'https://example.com/thumb.jpg', got '%s'", res.Cover)
	}
}

func TestDLsiteFetcher_Search_AuthorNarratorSplit_Grid(t *testing.T) {
	mockHTML := `
	<html><body>
		<ul class="n_worklist">
			<li>
				<div class="work_name"><a href="https://www.dlsite.com/maniax/work/=/product_id/RJ123456.html">Split Test Grid</a></div>
				<div class="maker_name">
					<a href="#">Circle Grid</a>
					<span class="separator">/</span>
					<span class="author"><a href="#">CV Grid</a></span>
				</div>
				<div class="work_thumb_inner">
					<img data-src="//example.com/grid_thumb.jpg" />
				</div>
			</li>
		</ul>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "fsr") || strings.Contains(r.URL.Path, "keyword") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "split test grid")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.Author != "Circle Grid" {
		t.Errorf("Expected Author 'Circle Grid', got '%s'", res.Author)
	}
	if res.Narrator != "CV Grid" {
		t.Errorf("Expected Narrator 'CV Grid', got '%s'", res.Narrator)
	}
	if res.Cover != "https://example.com/grid_thumb.jpg" {
		t.Errorf("Expected Cover 'https://example.com/grid_thumb.jpg', got '%s'", res.Cover)
	}
}
func TestDLsiteFetcher_Search_Keyword_FullMetadata(t *testing.T) {
	searchHTML := `
	<html><body>
		<table id="search_result_list">
			<tr>
				<td class="work_name"><a href="https://www.dlsite.com/maniax/work/=/product_id/RJ999999.html">Partial Title</a></td>
				<td class="maker_name"><a href="#">Partial Maker</a></td>
			</tr>
		</table>
	</body></html>`

	workHTML := `
	<html>
        <body>
            <h1 id="work_name">Full Title</h1>
            <span class="maker_name"><a href="#">Full Circle</a></span>
            <div class="product-slider-data">
                <div data-src="//example.com/full_cover.jpg"></div>
            </div>
            <table id="work_outline">
                <tr><th>販売日</th><td><a href="#">2023年12月31日</a></td></tr>
                <tr><th>ジャンル</th><td><a href="#">Full Tag</a></td></tr>
                <tr><th>声優</th><td><a href="#">Full Actor</a></td></tr>
            </table>
			<div class="work_parts_area">Full Description</div>
        </body>
    </html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/fsr/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(searchHTML))
			return
		}
		if strings.Contains(r.URL.Path, "/product_id/RJ999999.html") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(workHTML))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "keyword")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res := results[0]
	// Verify we got the FULL metadata, not the partial one
	if res.Title != "Full Title" {
		t.Errorf("Expected full title 'Full Title', got '%s'", res.Title)
	}
	if res.Author != "Full Circle" {
		t.Errorf("Expected full author 'Full Circle', got '%s'", res.Author)
	}
	if res.Narrator != "Full Actor" {
		t.Errorf("Expected full narrator 'Full Actor', got '%s'", res.Narrator)
	}
	if res.Cover != "https://example.com/full_cover.jpg" {
		t.Errorf("Expected full cover 'https://example.com/full_cover.jpg', got '%s'", res.Cover)
	}
}

func TestDLsiteFetcher_ExtractDescription_Fallback(t *testing.T) {
	mockHTML := `<html><head><meta property="og:description" content="Meta Description"></head><body></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	f := newTestFetcher(server.URL)

	results, err := f.Search(context.Background(), "RJ010101")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if results[0].Description != "Meta Description" {
		t.Errorf("Expected description 'Meta Description', got '%s'", results[0].Description)
	}
}

func TestDLsiteFetcher_ExplicitLogic(t *testing.T) {
	tests := []struct {
		rating   string
		expected bool
	}{
		{"全年齢", false},
		{"R-15", true},
		{"18禁", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.rating, func(t *testing.T) {
			f := &dlsiteFetcher{}
			work := AsmrWork{AgeRating: tt.rating}
			meta := f.toAbsMetadata(work)
			if meta.Explicit != tt.expected {
				t.Errorf("expected explicit %v for rating %q, got %v", tt.expected, tt.rating, meta.Explicit)
			}
		})
	}
}
func TestDLsiteFetcher_SeriesExtraction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/maniax/work/=/product_id/RJ999999.html", func(w http.ResponseWriter, r *http.Request) {
		html := `
			<h1 id="work_name">Test Work</h1>
			<span class="maker_name">Test Circle</span>
			<table id="work_outline">
				<tr><th>シリーズ名</th><td><a href="#">うちの子シリーズ</a></td></tr>
			</table>
			<div class="work_parts_area">Description</div>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	f := &dlsiteFetcher{
		client:  server.Client(),
		baseURL: server.URL,
	}

	rj, _ := NewRJCode("RJ999999")
	work, err := f.getWorkByID(context.Background(), rj)
	if err != nil {
		t.Fatalf("Failed to get work: %v", err)
	}

	if work.Series != "うちの子シリーズ" {
		t.Errorf("Expected Series 'うちの子シリーズ', got '%s'", work.Series)
	}
}

func TestDLsiteFetcher_SeriesExtraction_NoLink(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/maniax/work/=/product_id/RJ999998.html", func(w http.ResponseWriter, r *http.Request) {
		html := `
			<h1 id="work_name">Test Work</h1>
			<span class="maker_name">Test Circle</span>
			<table id="work_outline">
				<tr><th>シリーズ</th><td>Standalone Series</td></tr>
			</table>
			<div class="work_parts_area">Description</div>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	f := &dlsiteFetcher{
		client:  server.Client(),
		baseURL: server.URL,
	}

	rj, _ := NewRJCode("RJ999998")
	work, err := f.getWorkByID(context.Background(), rj)
	if err != nil {
		t.Fatalf("Failed to get work: %v", err)
	}

	if work.Series != "Standalone Series" {
		t.Errorf("Expected Series 'Standalone Series', got '%s'", work.Series)
	}
}
