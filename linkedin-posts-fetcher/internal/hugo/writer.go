// Package hugo generates Hugo-compatible markdown posts from LinkedIn data.
package hugo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const maxSlugLen = 60

// Post holds the parsed data needed to generate a Hugo markdown file.
type Post struct {
	Title        string
	Date         time.Time
	Body         string     // post text / commentary
	URL          string     // LinkedIn post URL
	AnalyticsURL string     // LinkedIn analytics URL
	ImageURLs    []string   // remote image URLs from LinkedIn CDN
	Likes        int        // 0 if unknown
	Views        int        // 0 if unknown
}

var (
	reNonAlnum    = regexp.MustCompile(`[^a-z0-9]+`)
	reMultiDash   = regexp.MustCompile(`-{2,}`)
)

// Slugify converts a title string into a URL-friendly slug matching the
// existing site convention: lowercase, non-alnum → hyphens, max 60 chars,
// trimmed on a hyphen boundary.
func Slugify(title string) string {
	s := strings.ToLower(title)

	// Normalize unicode (decompose accents etc.) and strip non-ASCII marks.
	s = norm.NFD.String(s)
	filtered := strings.Builder{}
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) { // Mn = non-spacing mark (accents)
			continue
		}
		filtered.WriteRune(r)
	}
	s = filtered.String()

	// Replace #word with "hashtag-word" (LinkedIn convention in existing posts)
	s = strings.ReplaceAll(s, "#", "hashtag-")

	// Replace non-alphanumeric with hyphens
	s = reNonAlnum.ReplaceAllString(s, "-")
	s = reMultiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	// Truncate to maxSlugLen, trying to break on a hyphen
	if len(s) > maxSlugLen {
		s = s[:maxSlugLen]
		// Don't chop mid-word if there's a hyphen nearby
		if idx := strings.LastIndex(s, "-"); idx > maxSlugLen-15 {
			s = s[:idx]
		}
	}

	return s
}

// Filename generates the markdown filename for a post: YYYY-MM-DD-slug.md
// If the title is empty, fallback is used (e.g. "linkedin-post-N").
func Filename(date time.Time, title, fallback string) string {
	slug := Slugify(title)
	if slug == "" {
		slug = fallback
	}
	return fmt.Sprintf("%s-%s.md", date.Format("2006-01-02"), slug)
}

// ImageFilename generates the image filename for a given post slug and
// image index. Indexes start at 2 to match the existing convention.
func ImageFilename(postFilename string, imageIndex int, ext string) string {
	// Strip .md from post filename
	base := strings.TrimSuffix(postFilename, ".md")
	return fmt.Sprintf("%s-image%d%s", base, imageIndex, ext)
}

// WritePost writes a Hugo markdown file to contentDir.
// It returns the filename written, or ("", nil) if the file already exists (skip).
func WritePost(contentDir string, post Post, fallbackSlug string) (string, error) {
	fname := Filename(post.Date, post.Title, fallbackSlug)
	path := filepath.Join(contentDir, fname)

	// Dedup: skip if file already exists
	if _, err := os.Stat(path); err == nil {
		return "", nil
	}

	var buf strings.Builder

	// --- Frontmatter ---
	buf.WriteString("+++\n")
	fmt.Fprintf(&buf, "title = %q\n", post.Title)
	fmt.Fprintf(&buf, "date = %q\n", post.Date.Format("2006-01-02T15:04:05"))
	buf.WriteString("draft = false\n")
	buf.WriteString("[params]\n")
	buf.WriteString("  source = \"linkedin\"\n")
	fmt.Fprintf(&buf, "  likes = %d\n", post.Likes)
	fmt.Fprintf(&buf, "  views = %d\n", post.Views)
	if post.URL != "" {
		fmt.Fprintf(&buf, "  url = %q\n", post.URL)
	}
	if post.AnalyticsURL != "" {
		fmt.Fprintf(&buf, "  linkedin_analytics_url = %q\n", post.AnalyticsURL)
	}
	if len(post.ImageURLs) > 0 {
		buf.WriteString("  image_sources = [")
		for i, u := range post.ImageURLs {
			if i > 0 {
				buf.WriteString(", ")
			}
			fmt.Fprintf(&buf, "%q", u)
		}
		buf.WriteString("]\n")
	}
	buf.WriteString("+++\n")

	// --- Body ---
	buf.WriteString("\n")
	if post.Body != "" {
		buf.WriteString(post.Body)
		buf.WriteString("\n")
	}

	// --- Images ---
	for i := range post.ImageURLs {
		imageIdx := i + 2 // existing convention: starts at 2
		ext := guessImageExt(post.ImageURLs[i])
		imgFile := ImageFilename(fname, imageIdx, ext)
		buf.WriteString("\n")
		fmt.Fprintf(&buf, "![Post image %d](/media/%s)\n", imageIdx, imgFile)
	}

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		return "", fmt.Errorf("creating content dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", fname, err)
	}

	return fname, nil
}

// guessImageExt returns an extension based on URL hints. Defaults to ".jpg".
func guessImageExt(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, ".png"):
		return ".png"
	case strings.Contains(lower, ".gif"):
		return ".gif"
	case strings.Contains(lower, ".webp"):
		return ".webp"
	default:
		return ".jpg"
	}
}
