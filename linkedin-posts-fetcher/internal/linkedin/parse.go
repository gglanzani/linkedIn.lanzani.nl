package linkedin

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lanzani/linkedin-posts-fetcher/internal/hugo"
)

// RichMediaItem represents an image/document from the RICH_MEDIA snapshot domain.
type RichMediaItem struct {
	DateTime    string // human-readable date string
	Description string
	MediaLink   string // CDN URL (may be empty)
	Date        time.Time
}

// ParseRichMedia extracts media items from the RICH_MEDIA snapshot domain.
func ParseRichMedia(items []map[string]interface{}) []RichMediaItem {
	var media []RichMediaItem
	var skippedOnce bool
	for _, item := range items {
		rm := RichMediaItem{
			DateTime:    getString(item, "Date/Time"),
			Description: getString(item, "Media Description"),
			MediaLink:   getString(item, "Media Link"),
		}
		// Skip entries with no usable media link
		if rm.MediaLink == "" {
			if !skippedOnce {
				// Log fields of the first skipped entry to discover alternative field names
				log.Printf("  [rich-media-debug] Skipped entry with no 'Media Link'. Fields present:")
				for key, val := range item {
					log.Printf("    %q = %v", key, truncateValue(val))
				}
				skippedOnce = true
			}
			continue
		}
		// Try to extract a date from the human-readable string
		// Format: "You uploaded a feed photo on January 29, 2026 at 8:30 AM (GMT)"
		rm.Date = parseFuzzyDate(rm.DateTime)
		media = append(media, rm)
	}
	return media
}

// MatchMediaToPost finds RICH_MEDIA items whose date is close to the post date.
// It first tries a tight tolerance (e.g. 5 minutes), then falls back to same-day
// matching if no tight matches are found. This handles cases where images are
// uploaded minutes or hours before the post is published.
func MatchMediaToPost(postDate time.Time, media []RichMediaItem, tightTolerance time.Duration) []string {
	var tight []string
	var sameDay []string

	postDay := postDate.Format("2006-01-02")

	// Track closest match for debug logging
	var closestDiff time.Duration
	var closestMedia *RichMediaItem
	var zeroDates int

	for i, m := range media {
		if m.Date.IsZero() {
			zeroDates++
			continue
		}
		diff := postDate.Sub(m.Date)
		if diff < 0 {
			diff = -diff
		}
		if closestMedia == nil || diff < closestDiff {
			closestDiff = diff
			closestMedia = &media[i]
		}
		if diff <= tightTolerance {
			tight = append(tight, m.MediaLink)
		}
		if m.Date.Format("2006-01-02") == postDay {
			sameDay = append(sameDay, m.MediaLink)
		}
	}

	if len(tight) > 0 {
		return tight
	}
	if len(sameDay) > 0 {
		return sameDay
	}

	// No match — log debug info to help diagnose
	if closestMedia != nil {
		log.Printf("    [media-debug] No media match for post %s. Closest: %s (diff=%v, raw=%q)",
			postDay, closestMedia.Date.Format("2006-01-02 15:04"), closestDiff, closestMedia.DateTime)
	}
	if zeroDates > 0 {
		log.Printf("    [media-debug] %d media items had unparseable dates", zeroDates)
	}

	return nil
}

// parseFuzzyDate extracts a date from LinkedIn's human-readable RICH_MEDIA date strings.
// Example: "You uploaded a feed photo on January 29, 2026 at 8:30 AM (GMT)"
func parseFuzzyDate(s string) time.Time {
	// Try to find "on <Month> <Day>, <Year> at <Time>" pattern
	// Strip the prefix up to " on "
	if idx := strings.Index(s, " on "); idx >= 0 {
		s = s[idx+4:]
	}
	// Strip timezone suffix like " (GMT)"
	if idx := strings.LastIndex(s, " ("); idx >= 0 {
		s = s[:idx]
	}
	// Try parsing: "January 29, 2026 at 8:30 AM"
	for _, layout := range []string{
		"January 2, 2006 at 3:04 PM",
		"January 2, 2006 at 3:04:05 PM",
		"January 02, 2006 at 3:04 PM",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ParseFuzzyDateExported is an exported wrapper around parseFuzzyDate for debugging.
func ParseFuzzyDateExported(s string) time.Time {
	return parseFuzzyDate(s)
}

// ParseSnapshotPost extracts a hugo.Post from a MEMBER_SHARE_INFO snapshot record.
// Returns nil if the record doesn't look like a post (e.g. missing commentary).
//
// Known MEMBER_SHARE_INFO fields (from LinkedIn Data Portability API):
//
//	Date             – "2026-02-05 19:11:15" (date string)
//	MediaUrl         – always empty (images are in RICH_MEDIA domain instead)
//	ShareCommentary  – the post text
//	ShareLink        – URL of the post on LinkedIn (URL-encoded)
//	SharedUrl        – URL shared in the post (if any)
//	Visibility       – e.g. "MEMBER_NETWORK"
func ParseSnapshotPost(item map[string]interface{}) *hugo.Post {
	commentary := getString(item, "ShareCommentary")
	shareLink := getString(item, "ShareLink")

	// URL-decode the share link (LinkedIn API returns encoded URNs like urn%3Ali%3Ashare%3A...)
	if shareLink != "" {
		if decoded, err := url.QueryUnescape(shareLink); err == nil {
			shareLink = decoded
		}
	}

	// Clean up LinkedIn's quote-wrapped paragraph encoding.
	// The API returns text like: First paragraph"\n""\n"Second paragraph
	// where quotes wrap each paragraph and "" acts as a separator.
	commentary = cleanLinkedInText(commentary)

	// Unfurl short URLs (lnkd.in, bit.ly, etc.) to their final destinations.
	commentary = unfurlURLs(commentary)

	// We need at least some text or a link to consider this a post
	if commentary == "" && shareLink == "" {
		return nil
	}

	post := &hugo.Post{
		Body: commentary,
	}

	// Build LinkedIn URL
	if shareLink != "" {
		post.URL = shareLink
	}

	// Try to extract date: first from explicit fields, then from the URN ID
	post.Date = extractDate(item)
	if post.Date.Equal(time.Time{}) || post.Date.After(time.Now().Add(24*time.Hour)) {
		// Fallback: try to extract timestamp from the LinkedIn URN (snowflake ID)
		if t := dateFromURN(shareLink); !t.IsZero() {
			post.Date = t
		} else {
			post.Date = time.Now()
		}
	}

	// Build analytics URL from share link if it contains a URN
	if shareLink != "" {
		if urn := extractURN(shareLink); urn != "" {
			// Convert share URN to activity URN for analytics
			activityURN := strings.Replace(urn, "urn:li:share:", "urn:li:activity:", 1)
			activityURN = strings.Replace(activityURN, "urn:li:ugcPost:", "urn:li:activity:", 1)
			post.AnalyticsURL = fmt.Sprintf("https://www.linkedin.com/analytics/post-summary/%s/", activityURN)
		}
	}

	// Title: human-readable date+time
	post.Title = post.Date.Format("January 2, 2006 at 3:04 PM")

	// Collect image URLs from known media fields
	for _, key := range []string{"ShareMediaUrl", "ShareMediaURL", "MediaUrl", "MediaURL", "ImageUrl", "ImageURL"} {
		if u := getString(item, key); u != "" && looksLikeImageURL(u) {
			post.ImageURLs = append(post.ImageURLs, u)
		}
	}

	// SharedUrl sometimes contains the image URL (e.g. when a post shares a photo)
	if sharedURL := getString(item, "SharedUrl"); sharedURL != "" && looksLikeImageURL(sharedURL) {
		post.ImageURLs = append(post.ImageURLs, sharedURL)
	}

	// Catch-all: scan all string values for anything that looks like an image URL
	// This helps discover image fields we haven't explicitly mapped yet.
	knownFields := map[string]bool{
		"ShareCommentary": true, "ShareLink": true, "MediaUrl": true,
		"SharedUrl": true, "Visibility": true, "Date": true,
	}
	for key, val := range item {
		if knownFields[key] {
			continue
		}
		// Log all unknown fields for discovery
		log.Printf("  [discovery] MEMBER_SHARE_INFO field: %q = %v", key, truncateValue(val))

		// If it looks like a media URL, add it as an image source
		if s, ok := val.(string); ok && looksLikeImageURL(s) {
			log.Printf("  [discovery] → treating %q as image URL", key)
			post.ImageURLs = append(post.ImageURLs, s)
		}
	}

	// Last resort: if no images found yet, try scraping the og:image from the
	// LinkedIn post page. This catches images that aren't in RICH_MEDIA.
	if len(post.ImageURLs) == 0 && post.URL != "" {
		if ogImg := scrapeOGImage(post.URL); ogImg != "" {
			post.ImageURLs = append(post.ImageURLs, ogImg)
			log.Printf("  🖼 Found og:image for %s", post.URL)
		}
	}

	return post
}

// ParseChangelogPost extracts a hugo.Post from a ChangelogEvent.
// Only processes CREATE events for shares/ugcPosts/articles.
// Returns nil if the event isn't a relevant post creation.
func ParseChangelogPost(evt ChangelogEvent) *hugo.Post {
	// Only interested in CREATE events for post-like resources
	if evt.Method != "CREATE" {
		return nil
	}

	switch evt.ResourceName {
	case "shares", "ugcPosts", "articles":
		// These are post types we care about
	default:
		return nil
	}

	if evt.ActivityStatus == "FAILURE" {
		return nil
	}

	// Prefer processedActivity; fall back to activity
	rawData := evt.ProcessedActivity
	if len(rawData) == 0 || string(rawData) == "null" {
		rawData = evt.Activity
	}

	if len(rawData) == 0 || string(rawData) == "null" {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rawData, &data); err != nil {
		log.Printf("  ⚠ Error parsing changelog activity JSON: %v", err)
		return nil
	}

	post := &hugo.Post{
		Date: time.UnixMilli(evt.CapturedAt),
	}

	// Extract text from various possible locations in the activity data
	post.Body = extractText(data)

	// Title: human-readable date+time
	post.Title = post.Date.Format("January 2, 2006 at 3:04 PM")

	// Build URL from resource ID
	if evt.ResourceID != "" {
		resourceID := string(evt.ResourceID)
		post.URL = fmt.Sprintf("https://www.linkedin.com/feed/update/%s", resourceID)

		// Build analytics URL
		activityURN := strings.Replace(resourceID, "urn:li:share:", "urn:li:activity:", 1)
		activityURN = strings.Replace(activityURN, "urn:li:ugcPost:", "urn:li:activity:", 1)
		post.AnalyticsURL = fmt.Sprintf("https://www.linkedin.com/analytics/post-summary/%s/", activityURN)
	}

	// Try to extract image URLs from activity data
	post.ImageURLs = extractImageURLs(data)

	// Log all top-level keys for discovery
	for key := range data {
		log.Printf("  [discovery] Changelog %s field: %q", evt.ResourceName, key)
	}

	return post
}

// --- helpers ---

func getString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func extractDate(item map[string]interface{}) time.Time {
	// Try various date field names
	for _, key := range []string{"Date", "CreatedAt", "CreatedTime", "SharedDate"} {
		if v, ok := item[key]; ok {
			switch val := v.(type) {
			case string:
				// Try parsing various formats
				for _, layout := range []string{
					time.RFC3339,
					"2006-01-02T15:04:05",
					"2006-01-02 15:04:05",
					"2006-01-02",
				} {
					if t, err := time.Parse(layout, val); err == nil {
						return t
					}
				}
			case float64:
				// Epoch ms
				if val > 1e12 {
					return time.UnixMilli(int64(val))
				}
				// Epoch seconds
				return time.Unix(int64(val), 0)
			}
		}
	}
	// Return zero time; caller should use fallback (e.g. URN-based date)
	return time.Time{}
}

func extractURN(shareLink string) string {
	// URL-decode first in case the link is percent-encoded
	decoded := shareLink
	if d, err := url.QueryUnescape(shareLink); err == nil {
		decoded = d
	}

	// LinkedIn URLs contain URNs like urn:li:share:12345 or urn:li:ugcPost:12345
	if idx := strings.Index(decoded, "urn:li:"); idx >= 0 {
		urn := decoded[idx:]
		// Trim any trailing path or query params
		for _, sep := range []string{"/", "?", "&"} {
			if i := strings.Index(urn, sep); i >= 0 {
				urn = urn[:i]
			}
		}
		return urn
	}
	return ""
}

// dateFromURN attempts to extract a timestamp from a LinkedIn URN's numeric ID.
// LinkedIn uses snowflake-like IDs where the upper bits encode a timestamp.
// The epoch offset and bit layout: timestamp_ms = id >> 22 (no custom epoch offset
// — LinkedIn IDs use Unix epoch directly, unlike Twitter which has a custom epoch).
//
// If the URL/URN doesn't contain a valid numeric ID, returns zero time.
func dateFromURN(shareLink string) time.Time {
	urn := extractURN(shareLink)
	if urn == "" {
		return time.Time{}
	}

	// Extract the numeric part after the last colon: "urn:li:share:12345" → "12345"
	parts := strings.Split(urn, ":")
	if len(parts) == 0 {
		return time.Time{}
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return time.Time{}
	}

	// LinkedIn snowflake: timestamp_ms = id >> 22
	tsMs := id >> 22
	if tsMs <= 0 {
		return time.Time{}
	}

	t := time.UnixMilli(tsMs)

	// Sanity check: should be between 2010 and next year
	if t.Year() < 2010 || t.Year() > time.Now().Year()+1 {
		return time.Time{}
	}

	return t
}

// extractText tries to find post text in various locations within the activity JSON.
func extractText(data map[string]interface{}) string {
	// UGC posts: specificContent.com.linkedin.ugc.ShareContent.shareCommentary.text
	if sc, ok := data["specificContent"].(map[string]interface{}); ok {
		for _, v := range sc {
			if content, ok := v.(map[string]interface{}); ok {
				if commentary, ok := content["shareCommentary"].(map[string]interface{}); ok {
					if text, ok := commentary["text"].(string); ok {
						return text
					}
				}
			}
		}
	}

	// Shares: commentary.text or text
	if commentary, ok := data["commentary"].(map[string]interface{}); ok {
		if text, ok := commentary["text"].(string); ok {
			return text
		}
	}

	if text, ok := data["text"].(string); ok {
		return text
	}

	if text, ok := data["commentary"].(string); ok {
		return text
	}

	return ""
}

// extractImageURLs tries to find image URLs in activity data.
func extractImageURLs(data map[string]interface{}) []string {
	var urls []string

	// UGC: specificContent.*.shareMediaCategory + shareContent.media[].originalUrl
	if sc, ok := data["specificContent"].(map[string]interface{}); ok {
		for _, v := range sc {
			if content, ok := v.(map[string]interface{}); ok {
				urls = append(urls, extractMediaFromContent(content)...)
			}
		}
	}

	// Shares: content.contentEntities[].entityLocation
	if content, ok := data["content"].(map[string]interface{}); ok {
		urls = append(urls, extractMediaFromContent(content)...)
	}

	// Direct media array
	if media, ok := data["media"].([]interface{}); ok {
		for _, m := range media {
			if mediaMap, ok := m.(map[string]interface{}); ok {
				for _, key := range []string{"originalUrl", "url", "downloadUrl"} {
					if u, ok := mediaMap[key].(string); ok && u != "" {
						urls = append(urls, u)
						break
					}
				}
			}
		}
	}

	return urls
}

func extractMediaFromContent(content map[string]interface{}) []string {
	var urls []string

	if media, ok := content["media"].([]interface{}); ok {
		for _, m := range media {
			if mediaMap, ok := m.(map[string]interface{}); ok {
				for _, key := range []string{"originalUrl", "url", "downloadUrl"} {
					if u, ok := mediaMap[key].(string); ok && u != "" {
						urls = append(urls, u)
						break
					}
				}
			}
		}
	}

	if entities, ok := content["contentEntities"].([]interface{}); ok {
		for _, e := range entities {
			if entity, ok := e.(map[string]interface{}); ok {
				if loc, ok := entity["entityLocation"].(string); ok && loc != "" {
					urls = append(urls, loc)
				}
			}
		}
	}

	return urls
}

// cleanLinkedInText removes LinkedIn's Data Portability API quote encoding.
// The API wraps paragraphs in quotes and uses empty quoted lines as separators:
//
//	First paragraph"
//	""
//	"Second paragraph"
//	""
//	"Third paragraph
//
// This function strips the quotes and normalizes to plain paragraphs separated
// by double newlines.
func cleanLinkedInText(s string) string {
	if s == "" {
		return ""
	}

	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, line := range lines {
		// Strip leading/trailing quotes from each line
		line = strings.TrimSpace(line)

		// Skip empty-quote lines that serve as paragraph separators
		if line == `""` || line == `"` {
			// Add a blank line to preserve paragraph breaks
			cleaned = append(cleaned, "")
			continue
		}

		// Strip leading quote
		if strings.HasPrefix(line, `"`) {
			line = line[1:]
		}
		// Strip trailing quote
		if strings.HasSuffix(line, `"`) {
			line = line[:len(line)-1]
		}

		cleaned = append(cleaned, line)
	}

	result := strings.Join(cleaned, "\n")

	// Collapse 3+ consecutive newlines into 2 (paragraph break)
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(result)
}

// looksLikeImageURL checks if a string looks like a URL pointing to an image.
func looksLikeImageURL(s string) bool {
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	// LinkedIn CDN or common image patterns
	imageHints := []string{
		"media.licdn.com",
		"media-exp", // LinkedIn media CDN variants
		".jpg", ".jpeg", ".png", ".gif", ".webp",
		"/image/", "/photo/", "/media/",
	}
	for _, hint := range imageHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// ogImagePattern matches the og:image meta tag in LinkedIn post pages.
var ogImagePattern = regexp.MustCompile(`<meta\s+(?:property|name)="og:image"\s+content="([^"]+)"`)

// scrapeOGImage fetches a LinkedIn post page and extracts the og:image URL.
// Returns empty string if the page can't be fetched or has no og:image.
func scrapeOGImage(postURL string) string {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(postURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	// Only read the first 100KB — og:image is always in the <head>
	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return ""
	}

	matches := ogImagePattern.FindSubmatch(body)
	if len(matches) < 2 {
		return ""
	}

	imgURL := strings.ReplaceAll(string(matches[1]), "&amp;", "&")

	// Verify it's actually an image URL (not a default LinkedIn placeholder)
	if strings.Contains(imgURL, "media.licdn.com") {
		return imgURL
	}

	return ""
}

// shortURLPattern matches common URL shortener domains found in LinkedIn posts.
var shortURLPattern = regexp.MustCompile(`https?://(?:lnkd\.in|bit\.ly|t\.co|goo\.gl|ow\.ly|buff\.ly|is\.gd|tinyurl\.com)/[^\s)\]]+`)

// interstitialHrefPattern extracts the destination URL from LinkedIn's
// interstitial "external link" page. The page contains an <a> tag with
// data-tracking-control-name="external_url_click" and the real URL in href.
var interstitialHrefPattern = regexp.MustCompile(`href="(https?://[^"]+)"[^>]*>\s*https?://`)

// unfurlURLs finds short URLs in text and replaces them with their final
// destination. LinkedIn's lnkd.in links don't use HTTP redirects — they
// serve an interstitial HTML page with the real URL embedded. This function
// fetches the page and extracts the destination. For non-LinkedIn shorteners,
// it follows HTTP redirects. URLs that fail to resolve are left unchanged.
func unfurlURLs(text string) string {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil // follow redirects
		},
	}

	return shortURLPattern.ReplaceAllStringFunc(text, func(shortURL string) string {
		resp, err := client.Get(shortURL)
		if err != nil {
			log.Printf("  ⚠ Could not unfurl %s: %v", shortURL, err)
			return shortURL
		}
		defer resp.Body.Close()

		// If we followed redirects to a different host, use the final URL
		finalURL := resp.Request.URL.String()
		if finalURL != shortURL && resp.Request.URL.Host != "lnkd.in" {
			log.Printf("  🔗 Unfurled %s → %s", shortURL, finalURL)
			return finalURL
		}

		// LinkedIn's lnkd.in serves an interstitial page — parse the HTML
		// to extract the external destination URL.
		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // cap at 512KB
		if err != nil {
			log.Printf("  ⚠ Could not read %s body: %v", shortURL, err)
			return shortURL
		}

		// Look for the external link in the interstitial page
		matches := interstitialHrefPattern.FindSubmatch(body)
		if len(matches) >= 2 {
			dest := string(matches[1])
			// Skip LinkedIn-internal links (they're not useful unfurled)
			if !strings.Contains(dest, "linkedin.com") && !strings.Contains(dest, "licdn.com") {
				log.Printf("  🔗 Unfurled %s → %s", shortURL, dest)
				return dest
			}
		}

		// Fallback: look for any non-LinkedIn external href
		allHrefs := regexp.MustCompile(`href="(https?://[^"]+)"`).FindAllSubmatch(body, -1)
		for _, m := range allHrefs {
			href := string(m[1])
			if !strings.Contains(href, "linkedin.com") && !strings.Contains(href, "licdn.com") {
				log.Printf("  🔗 Unfurled %s → %s", shortURL, href)
				return href
			}
		}

		log.Printf("  ⚠ Could not extract destination from %s (status %d)", shortURL, resp.StatusCode)
		return shortURL
	})
}

func truncateValue(v interface{}) string {
	s := fmt.Sprintf("%v", v)
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
