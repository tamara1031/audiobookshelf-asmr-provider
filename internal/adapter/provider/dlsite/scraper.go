package dlsite

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"audiobookshelf-asmr-provider/internal/domain"

	"github.com/PuerkitoBio/goquery"
)

type dlsiteFetcher struct {
	client  *http.Client
	baseURL string
}

// NewDLsiteFetcher creates a new instance of the DLsite provider.
func NewDLsiteFetcher() domain.Provider {
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
func (f *dlsiteFetcher) Search(ctx context.Context, query string) ([]domain.AbsBookMetadata, error) {
	if rj, err := NewRJCode(query); err == nil {
		work, err := f.getWorkByID(ctx, rj)
		if err != nil {
			return nil, err
		}
		return []domain.AbsBookMetadata{f.toAbsMetadata(work)}, nil
	}
	// Keyword search implementation would go here
	return nil, nil
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

func (f *dlsiteFetcher) toAbsMetadata(work AsmrWork) domain.AbsBookMetadata {
	return domain.AbsBookMetadata{
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
