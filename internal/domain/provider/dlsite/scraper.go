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
	doc.Find("#search_result_list tr").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".work_name a").Text())
		link, _ := s.Find(".work_name a").Attr("href")
		// Extract maker and narrator using classes
		makerElem := s.Find(".maker_name")
		// The circle is usually the first link, direct child or just before separation
		// Ideally we select direct child, but goquery's Find parses descendants.
		// We can filter out the narrator link if we find it separately.

		var maker, narrator string

		// Attempt to identify narrator explicitly
		narratorElem := makerElem.Find(".author a")
		if narratorElem.Length() > 0 {
			narrator = strings.TrimSpace(narratorElem.Text())
		}

		// For maker, we might get everything if we just do Text().
		// If we use .maker_name > a, we get the circle.
		// However, goquery's selector support might be limited.
		// Let's try to find the first anchor, check if it matches narrator, if not it's circle.
		makerElem.Find("a").Each(func(i int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			// If this text is the narrator, skip (unless circle and narrator same? unlikely)
			if text == narrator {
				return
			}
			// First non-narrator link is likely the circle
			if maker == "" {
				maker = text
			}
		})

		if title == "" {
			return
		}

		// Extract RJ code from link
		var rjCode string
		if found := extractor.FindString(link); found != "" {
			rjCode = found
		} else {
			// Fallback
			if strings.Contains(link, "RJ") {
				rjCode = extractor.FindString(link)
			}
		}

		if title != "" && rjCode != "" {
			results = append(results, service.AbsBookMetadata{
				Title:     title,
				Author:    maker,
				Narrator:  narrator,
				ISBN:      rjCode,
				Publisher: "DLsite",
				Explicit:  true,
				Language:  "Japanese",
			})
		}
	})

	// If no results from table, try grid format (n_worklist)
	if len(results) == 0 {
		doc.Find(".n_worklist li").Each(func(i int, s *goquery.Selection) {

			title := strings.TrimSpace(s.Find(".work_name a").Text())
			link, _ := s.Find(".work_name a").Attr("href")
			// Extract maker and narrator using classes
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

			// Extract RJ code from link
			var rjCode string
			if found := extractor.FindString(link); found != "" {
				rjCode = found
			} else {
				// Try to extract from the end of URL if regex didn't match directly
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

			if title != "" && rjCode != "" {
				results = append(results, service.AbsBookMetadata{
					Title:     title,
					Author:    maker,
					Narrator:  narrator,
					ISBN:      rjCode,
					Publisher: "DLsite",
					Explicit:  true,
					Language:  "Japanese",
					// We could extract more info here (cover, etc.) but let's start with basic
				})
			}
		})
	}

	return results, nil
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
