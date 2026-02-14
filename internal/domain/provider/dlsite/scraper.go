package dlsite

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"audiobookshelf-asmr-provider/internal/service"
)

type dlsiteFetcher struct {
	client  *http.Client
	baseURL string
}

// NewDLsiteFetcher creates a new instance of the DLsite provider.
func NewDLsiteFetcher() service.Provider {
	return &dlsiteFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://www.dlsite.com",
	}
}

// ID returns the unique identifier for this provider.
func (f *dlsiteFetcher) ID() string {
	return "dlsite"
}

// CacheTTL returns the cache duration for this provider (24 hours).
func (f *dlsiteFetcher) CacheTTL() time.Duration {
	return 24 * time.Hour
}

// Search searches for works matching the query. Currently only supports RJ codes.
func (f *dlsiteFetcher) Search(ctx context.Context, query string) ([]service.AbsBookMetadata, error) {
	if rj, err := NewRJCode(query); err == nil {
		work, err := f.getWorkByID(ctx, rj)
		if err != nil {
			return nil, err
		}
		return []service.AbsBookMetadata{f.toAbsMetadata(work)}, nil
	}
	// Keyword search implementation
	return f.searchKeywords(ctx, query)
}

func (f *dlsiteFetcher) searchKeywords(ctx context.Context, query string) ([]service.AbsBookMetadata, error) {
	searchURL := fmt.Sprintf("%s/maniax/fsr/=/keyword/%s", f.baseURL, url.QueryEscape(query))

	doc, err := f.fetchPage(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var results []service.AbsBookMetadata
	extractor := regexp.MustCompile(`(?i)RJ\d{6,8}`)

	// Try table format first (classic)
	doc.Find("#search_result_list tr").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if len(results) >= 5 {
			return false
		}
		if meta, ok := f.extractFromTable(s, extractor); ok {
			results = append(results, meta)
		}
		return true
	})

	// If no results from table, try grid format (n_worklist)
	if len(results) == 0 {
		doc.Find(".n_worklist li").EachWithBreak(func(i int, s *goquery.Selection) bool {
			if len(results) >= 5 {
				return false
			}
			if meta, ok := f.extractFromGrid(s, extractor); ok {
				results = append(results, meta)
			}
			return true
		})
	}

	// Enhance results with full metadata
	for i, res := range results {
		if res.ISBN == "" {
			continue
		}
		rjCode, err := NewRJCode(res.ISBN)
		if err != nil {
			continue
		}

		// Fetch full details
		work, err := f.getWorkByID(ctx, rjCode)
		if err == nil {
			results[i] = f.toAbsMetadata(work)
		}
		// If error, keep the partial result from search page
	}

	return results, nil
}

func (f *dlsiteFetcher) extractFromTable(s *goquery.Selection, extractor *regexp.Regexp) (service.AbsBookMetadata, bool) {
	title := strings.TrimSpace(s.Find(".work_name a").Text())
	if title == "" {
		return service.AbsBookMetadata{}, false
	}
	link, _ := s.Find(".work_name a").Attr("href")

	maker, narrator := f.extractMakerAndNarrator(s)

	var rjCode string
	if found := extractor.FindString(link); found != "" {
		rjCode = found
	} else if strings.Contains(link, "RJ") {
		rjCode = extractor.FindString(link)
	}

	if rjCode == "" {
		return service.AbsBookMetadata{}, false
	}

	var coverURL string
	img := s.Find(".search_result_img_box_inner img")
	if src, exists := img.Attr("src"); exists {
		coverURL = src
	}
	if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" {
		coverURL = dataSrc
	}
	if coverURL != "" && strings.HasPrefix(coverURL, "//") {
		coverURL = "https:" + coverURL
	}

	return service.AbsBookMetadata{
		Title:     title,
		Author:    maker,
		Narrator:  narrator,
		ISBN:      rjCode,
		Publisher: "DLsite",
		Explicit:  true,
		Language:  "Japanese",
		Cover:     coverURL,
	}, true
}

func (f *dlsiteFetcher) extractFromGrid(s *goquery.Selection, extractor *regexp.Regexp) (service.AbsBookMetadata, bool) {
	title := strings.TrimSpace(s.Find(".work_name a").Text())
	if title == "" {
		return service.AbsBookMetadata{}, false
	}
	link, _ := s.Find(".work_name a").Attr("href")

	maker, narrator := f.extractMakerAndNarrator(s)

	var rjCode string
	if found := extractor.FindString(link); found != "" {
		rjCode = found
	} else {
		parts := strings.Split(link, "/")
		for _, p := range parts {
			if strings.HasPrefix(strings.ToUpper(p), "RJ") {
				if strings.HasSuffix(p, ".html") {
					rjCode = strings.TrimSuffix(p, ".html")
				} else {
					rjCode = p
				}
				break
			}
		}
	}

	if rjCode == "" {
		return service.AbsBookMetadata{}, false
	}

	var coverURL string
	img := s.Find(".work_thumb_inner img")
	if src, exists := img.Attr("src"); exists {
		coverURL = src
	}
	if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" {
		coverURL = dataSrc
	}
	if coverURL != "" && strings.HasPrefix(coverURL, "//") {
		coverURL = "https:" + coverURL
	}

	return service.AbsBookMetadata{
		Title:     title,
		Author:    maker,
		Narrator:  narrator,
		ISBN:      rjCode,
		Publisher: "DLsite",
		Explicit:  true,
		Language:  "Japanese",
		Cover:     coverURL,
	}, true
}

func (f *dlsiteFetcher) extractMakerAndNarrator(s *goquery.Selection) (string, string) {
	makerElem := s.Find(".maker_name")
	var maker, narrator string

	narratorElem := makerElem.Find(".author a")
	if narratorElem.Length() > 0 {
		narrator = strings.TrimSpace(narratorElem.Text())
	}

	makerElem.Find("a").Each(func(i int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text == narrator {
			return
		}
		if maker == "" {
			maker = text
		}
	})
	return maker, narrator
}

// getWorkByID fetches and parses the work page for a given RJ code.
func (f *dlsiteFetcher) getWorkByID(ctx context.Context, code RJCode) (AsmrWork, error) {
	targetURL := fmt.Sprintf("%s/maniax/work/=/product_id/%s.html", f.baseURL, code.String())

	doc, err := f.fetchPage(ctx, targetURL)
	if err != nil {
		return AsmrWork{}, err
	}

	work := AsmrWork{
		RJCode:    code,
		DLsiteURL: targetURL,
		Title:     f.extractTitle(doc),
		Circle:    f.extractCircle(doc),
		CoverURL:  f.extractCoverURL(doc),
	}

	f.extractTableData(doc, &work)

	return work, nil
}

func (f *dlsiteFetcher) fetchPage(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{Name: "adult_checked", Value: "1"})
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("dlsite returned status: %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func (f *dlsiteFetcher) extractTitle(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find("#work_name").Text())
}

func (f *dlsiteFetcher) extractCircle(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find("span.maker_name a").Text())
}

func (f *dlsiteFetcher) extractCoverURL(doc *goquery.Document) string {
	imgNode := doc.Find(".product_slider_data div").First()
	imgSrc, exists := imgNode.Attr("data-src")
	if !exists {
		imgSrc, _ = imgNode.Attr("src")
	}
	if imgSrc != "" && strings.HasPrefix(imgSrc, "//") {
		return "https:" + imgSrc
	}
	return imgSrc
}

func (f *dlsiteFetcher) extractTableData(doc *goquery.Document, work *AsmrWork) {
	doc.Find("#work_outline tr").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Find("th").Text())
		data := s.Find("td")

		if strings.Contains(header, "声優") {
			data.Find("a").Each(func(_ int, a *goquery.Selection) {
				work.CV = append(work.CV, strings.TrimSpace(a.Text()))
			})
		} else if strings.Contains(header, "ジャンル") {
			data.Find("a").Each(func(_ int, a *goquery.Selection) {
				work.Tags = append(work.Tags, strings.TrimSpace(a.Text()))
			})
		} else if strings.Contains(header, "販売日") {
			dateStr := strings.TrimSpace(data.Find("a").Text())
			dateStr = strings.ReplaceAll(dateStr, "年", "-")
			dateStr = strings.ReplaceAll(dateStr, "月", "-")
			dateStr = strings.ReplaceAll(dateStr, "日", "")
			work.ReleaseDate = dateStr
		}
	})
}

func (f *dlsiteFetcher) toAbsMetadata(work AsmrWork) service.AbsBookMetadata {
	return service.AbsBookMetadata{
		Title:         work.Title,
		Author:        work.Circle,
		Narrator:      strings.Join(work.CV, ", "),
		Description:   work.Description,
		PublishedYear: work.ReleaseDate,
		Genres:        work.Tags,
		Tags:          work.Tags,
		Cover:         work.CoverURL,
		ISBN:          work.RJCode.String(),
		Explicit:      true,
		Language:      "Japanese",
		Publisher:     "DLsite",
	}
}
