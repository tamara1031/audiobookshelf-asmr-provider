package dlsite

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"audiobookshelf-asmr-provider/internal/service"
)

type dlsiteFetcher struct {
	client           *http.Client
	baseURL          string
	ageCheckDisabled bool
}

// NewDLsiteFetcher creates a new instance of the DLsite provider.
func NewDLsiteFetcher() service.Provider {
	disableAgeCheck := false
	ageCheckEnv := strings.ToLower(os.Getenv("DISABLE_AGE_CHECK"))
	if ageCheckEnv == "1" || ageCheckEnv == "true" || ageCheckEnv == "yes" {
		disableAgeCheck = true
	}

	return &dlsiteFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:          "https://www.dlsite.com",
		ageCheckDisabled: disableAgeCheck,
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
		RJCode:      code,
		DLsiteURL:   targetURL,
		Title:       f.extractTitle(doc),
		Circle:      f.extractCircle(doc),
		CoverURL:    f.extractCoverURL(doc),
		Description: f.extractDescription(doc), // Description抽出を追加
	}

	// テーブルデータ（声優、ジャンル、シリーズ、シナリオ、形式、年齢）を一括取得
	f.extractTableData(doc, &work)

	return work, nil
}

func (f *dlsiteFetcher) fetchPage(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if f.ageCheckDisabled {
		req.AddCookie(&http.Cookie{Name: "adult_checked", Value: "1"})
	}
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

// extractDescription: 作品内容（あらすじ）を抽出。<br>を改行に変換して可読性を維持
func (f *dlsiteFetcher) extractDescription(doc *goquery.Document) string {
	// 作品内容の主要エリア
	// ※通常は .work_parts_area だが、作品によっては .work_parts_type_text の中にある場合もあるため
	//   最も確実な .work_parts_area をターゲットにします
	selection := doc.Find(".work_parts_area").First()

	if selection.Length() == 0 {
		// 見つからない場合はmeta descriptionから取得（フォールバック）
		return strings.TrimSpace(doc.Find(`meta[property="og:description"]`).AttrOr("content", ""))
	}

	// HTMLを取得して <br> を改行コードに置換
	html, _ := selection.Html()
	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "<br/>", "\n")
	html = strings.ReplaceAll(html, "<br />", "\n")

	// タグを除去してテキストのみにする（簡易的なタグ除去）
	// 注意: 厳密なサニタイズが必要な場合は bluemonday などのライブラリ推奨ですが、
	// ここでは標準的な文字列置換とgoqueryのText()再パースで対応します
	tmpDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	return strings.TrimSpace(tmpDoc.Text())
}

func (f *dlsiteFetcher) extractCircle(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find("span.maker_name a").Text())
}

func (f *dlsiteFetcher) extractCoverURL(doc *goquery.Document) string {
	// 修正: アンダースコア(_) ではなく ハイフン(-) です
	imgNode := doc.Find(".product-slider-data div").First()

	imgSrc, exists := imgNode.Attr("data-src")
	if !exists {
		imgSrc, _ = imgNode.Attr("src")
	}

	// URLが見つかった場合の処理
	if imgSrc != "" {
		if strings.HasPrefix(imgSrc, "//") {
			return "https:" + imgSrc
		}
		return imgSrc
	}

	return ""
}

// extractTableData: テーブル情報から各フィールドへマッピング
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
			// 年月日フォーマットの整形
			dateStr = strings.ReplaceAll(dateStr, "年", "-")
			dateStr = strings.ReplaceAll(dateStr, "月", "-")
			dateStr = strings.ReplaceAll(dateStr, "日", "")
			work.ReleaseDate = dateStr
		} else {
			// 補助関数: td内のテキストを取得。リンクがある場合はそのテキストを優先
			getText := func(d *goquery.Selection) string {
				if a := d.Find("a"); a.Length() > 0 {
					return strings.TrimSpace(a.First().Text())
				}
				return strings.TrimSpace(d.Text())
			}

			if strings.Contains(header, "シリーズ") {
				work.Series = getText(data)
			} else if strings.Contains(header, "シナリオ") {
				work.Scenario = getText(data)
			} else if strings.Contains(header, "作品形式") {
				work.WorkFormat = getText(data)
			} else if strings.Contains(header, "年齢指定") {
				work.AgeRating = getText(data)
			}
		}
	})
}

// toAbsMetadata: AsmrWork から AbsBookMetadata への変換ロジック
func (f *dlsiteFetcher) toAbsMetadata(work AsmrWork) service.AbsBookMetadata {
	// Explicit判定: 年齢指定に「全年齢」が含まれていなければ true (R18など)
	isExplicit := !strings.Contains(work.AgeRating, "全年齢")

	// Author: 基本は「シナリオ」。もし空なら「サークル名」をフォールバックとして使用
	author := work.Scenario
	if author == "" {
		author = work.Circle
	}

	// Genres: 「作品形式」を格納
	var genres []string
	if work.WorkFormat != "" {
		genres = []string{work.WorkFormat}
	}

	// Series: ABS仕様に合わせてオブジェクト配列に変換
	var series []service.SeriesMetadata
	if work.Series != "" {
		series = []service.SeriesMetadata{
			{Series: work.Series},
		}
	}

	// PublishedYear: シリーズ・出版年として「年（YYYY）」のみを抽出（ABSの互換性重視）
	// YYYY-MM-DD から最初に向かって4文字取得
	year := work.ReleaseDate
	if len(year) >= 4 {
		year = year[:4]
	}

	return service.AbsBookMetadata{
		Title:         work.Title,
		Author:        author,
		Narrator:      strings.Join(work.CV, ", "),
		Series:        series,
		Description:   work.Description,
		Publisher:     work.Circle,
		PublishedYear: year,
		Genres:        genres,
		Tags:          work.Tags,
		Cover:         work.CoverURL,
		ISBN:          work.RJCode.String(),
		Explicit:      isExplicit,
		Language:      "Japanese",
	}
}
