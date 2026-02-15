// linkedin-posts-fetcher fetches your LinkedIn posts using the Member Data
// Portability APIs and generates Hugo-compatible markdown files.
//
// Usage:
//
//	linkedin-posts-fetcher generate  # fetch posts from LinkedIn → write Hugo markdown files
//	linkedin-posts-fetcher images    # download missing images referenced in posts
//	linkedin-posts-fetcher status    # check API authorization status
//
// Required environment variables:
//
//	LINKEDIN_ACCESS_TOKEN – OAuth token with r_dma_portability_member scope
//
// Optional environment variables:
//
//	CONTENT_DIR – path to Hugo content/posts dir (default: ../content/posts)
//	MEDIA_DIR   – path to Hugo static/media dir (default: ../static/media)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lanzani/linkedin-posts-fetcher/internal/hugo"
	"github.com/lanzani/linkedin-posts-fetcher/internal/linkedin"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		token := requireToken()
		client := linkedin.NewClient(token)
		contentDir, _ := resolveDirs()
		runGenerate(client, contentDir)

	case "images":
		contentDir, mediaDir := resolveDirs()
		runImages(contentDir, mediaDir)

	case "status":
		token := requireToken()
		client := linkedin.NewClient(token)
		runStatus(client)

	case "dump":
		token := requireToken()
		client := linkedin.NewClient(token)
		domain := "MEMBER_SHARE_INFO"
		if len(os.Args) > 2 {
			domain = os.Args[2]
		}
		dateFilter := ""
		if len(os.Args) > 3 {
			dateFilter = os.Args[3]
		}
		runDump(client, domain, dateFilter)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: linkedin-posts-fetcher <command>

Commands:
  generate   Fetch posts from LinkedIn and write Hugo markdown files
  images     Download missing images referenced in post markdown files
  status     Check LinkedIn API authorization status
  dump       Dump raw JSON from a snapshot domain (default: MEMBER_SHARE_INFO)

Environment:
  LINKEDIN_ACCESS_TOKEN  OAuth token (required for generate, status)
  CONTENT_DIR            Hugo content/posts directory (default: ../content/posts)
  MEDIA_DIR              Hugo static/media directory (default: ../static/media)
`)
}

func requireToken() string {
	token := os.Getenv("LINKEDIN_ACCESS_TOKEN")
	if token == "" {
		log.Fatal("LINKEDIN_ACCESS_TOKEN environment variable is required")
	}
	return token
}

func resolveDirs() (contentDir, mediaDir string) {
	contentDir = os.Getenv("CONTENT_DIR")
	if contentDir == "" {
		// Default: relative to the binary's location
		exe, _ := os.Executable()
		base := filepath.Dir(exe)
		contentDir = filepath.Join(base, "..", "content", "posts")
	}

	mediaDir = os.Getenv("MEDIA_DIR")
	if mediaDir == "" {
		exe, _ := os.Executable()
		base := filepath.Dir(exe)
		mediaDir = filepath.Join(base, "..", "static", "media")
	}

	return contentDir, mediaDir
}

// ---------------------------------------------------------------------------
// generate – fetch posts and write Hugo markdown
// ---------------------------------------------------------------------------

func runGenerate(client *linkedin.Client, contentDir string) {
	log.Println("=== Generating Hugo posts from LinkedIn ===")
	log.Printf("Content dir: %s", contentDir)

	// First, fetch RICH_MEDIA to build a media lookup (images live here, not in MEMBER_SHARE_INFO)
	var richMedia []linkedin.RichMediaItem
	log.Println("Fetching RICH_MEDIA from Snapshot API...")
	rmData, err := client.FetchAllSnapshotPages("RICH_MEDIA")
	if err != nil {
		log.Printf("⚠ Error fetching RICH_MEDIA: %v (images won't be matched)", err)
	} else {
		richMedia = linkedin.ParseRichMedia(rmData)
		log.Printf("Found %d media items with valid links (out of %d total)", len(richMedia), len(rmData))
	}

	created := 0
	skipped := 0
	errors := 0

	// Fetch MEMBER_SHARE_INFO (the main domain for your own posts)
	log.Println("Fetching MEMBER_SHARE_INFO from Snapshot API...")
	data, err := client.FetchAllSnapshotPages("MEMBER_SHARE_INFO")
	if err != nil {
		log.Printf("⚠ Error fetching MEMBER_SHARE_INFO: %v", err)
	} else {
		log.Printf("Received %d snapshot records", len(data))

		for i, item := range data {
			post := linkedin.ParseSnapshotPost(item)
			if post == nil {
				continue
			}

			// Match images from RICH_MEDIA by date proximity (within 5 minutes)
			if len(post.ImageURLs) == 0 && len(richMedia) > 0 {
				matched := linkedin.MatchMediaToPost(post.Date, richMedia, 5*time.Minute)
				if len(matched) > 0 {
					post.ImageURLs = matched
					log.Printf("  📎 Matched %d image(s) for post at %s", len(matched), post.Date.Format("2006-01-02 15:04"))
				}
			}

			fallback := fmt.Sprintf("linkedin-post-%d", i+1)
			fname, err := hugo.WritePost(contentDir, *post, fallback)
			if err != nil {
				log.Printf("  ⚠ Error writing post: %v", err)
				errors++
				continue
			}
			if fname == "" {
				skipped++
			} else {
				created++
				log.Printf("  + %s", fname)
			}
		}
	}

	// Also try ARTICLES for long-form posts
	log.Println("Fetching ARTICLES from Snapshot API...")
	articles, err := client.FetchAllSnapshotPages("ARTICLES")
	if err != nil {
		log.Printf("⚠ Error fetching ARTICLES: %v (may be empty)", err)
	} else {
		log.Printf("Received %d article records", len(articles))
		for i, item := range articles {
			post := linkedin.ParseSnapshotPost(item)
			if post == nil {
				continue
			}

			fallback := fmt.Sprintf("linkedin-article-%d", i+1)
			fname, err := hugo.WritePost(contentDir, *post, fallback)
			if err != nil {
				log.Printf("  ⚠ Error writing article: %v", err)
				errors++
				continue
			}
			if fname == "" {
				skipped++
			} else {
				created++
				log.Printf("  + %s", fname)
			}
		}
	}

	log.Printf("=== Done. Created: %d, Skipped (existing): %d, Errors: %d ===",
		created, skipped, errors)
}

// ---------------------------------------------------------------------------
// images – download missing post images
// ---------------------------------------------------------------------------

func runImages(contentDir, mediaDir string) {
	log.Println("=== Downloading missing images ===")
	log.Printf("Content dir: %s", contentDir)
	log.Printf("Media dir:   %s", mediaDir)

	if err := hugo.DownloadMissingImages(contentDir, mediaDir); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// status – check authorization
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// dump – output raw JSON from a snapshot domain for debugging
// ---------------------------------------------------------------------------

// runDump prints raw JSON from a snapshot domain.
// Usage: dump [DOMAIN] [DATE-FILTER]
//
//	dump                        → first 3 MEMBER_SHARE_INFO records
//	dump RICH_MEDIA             → first 3 RICH_MEDIA records
//	dump RICH_MEDIA 2018-06-29  → all RICH_MEDIA records near that date
func runDump(client *linkedin.Client, domain, dateFilter string) {
	log.Printf("=== Dumping raw snapshot data for domain: %s ===", domain)

	data, err := client.FetchAllSnapshotPages(domain)
	if err != nil {
		log.Fatalf("Error fetching %s: %v", domain, err)
	}

	log.Printf("Received %d records", len(data))

	// If a date filter is given and domain is RICH_MEDIA, show matching + nearby entries
	if dateFilter != "" && domain == "RICH_MEDIA" {
		filterDate, err := time.Parse("2006-01-02", dateFilter)
		if err != nil {
			log.Fatalf("Invalid date filter %q: %v", dateFilter, err)
		}

		richMedia := linkedin.ParseRichMedia(data)
		log.Printf("Parsed %d media items with valid links", len(richMedia))

		// Also count unparseable dates
		var zeroDates int
		for _, m := range richMedia {
			if m.Date.IsZero() {
				zeroDates++
			}
		}
		if zeroDates > 0 {
			log.Printf("⚠ %d media items have unparseable dates", zeroDates)
		}

		// Show all entries within ±3 days of the filter date
		fmt.Printf("\n--- RICH_MEDIA entries near %s ---\n", dateFilter)
		found := 0
		for _, m := range richMedia {
			if m.Date.IsZero() {
				continue
			}
			diff := filterDate.Sub(m.Date)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 72*time.Hour {
				found++
				fmt.Printf("  %s  %s\n    raw date: %q\n    link: %s\n",
					m.Date.Format("2006-01-02 15:04"), m.Description, m.DateTime, m.MediaLink)
			}
		}
		if found == 0 {
			fmt.Println("  (none found within ±3 days)")
		}

		// Also show the 3 entries with dates closest to the filter
		fmt.Printf("\n--- Closest RICH_MEDIA entries to %s ---\n", dateFilter)
		type distEntry struct {
			diff  time.Duration
			media linkedin.RichMediaItem
		}
		var closest []distEntry
		for _, m := range richMedia {
			if m.Date.IsZero() {
				continue
			}
			d := filterDate.Sub(m.Date)
			if d < 0 {
				d = -d
			}
			if len(closest) < 3 {
				closest = append(closest, distEntry{d, m})
			} else {
				// Replace the farthest if this is closer
				maxIdx := 0
				for j := 1; j < len(closest); j++ {
					if closest[j].diff > closest[maxIdx].diff {
						maxIdx = j
					}
				}
				if d < closest[maxIdx].diff {
					closest[maxIdx] = distEntry{d, m}
				}
			}
		}
		for _, c := range closest {
			fmt.Printf("  %s (diff=%v)  %s\n    raw date: %q\n    link: %s\n",
				c.media.Date.Format("2006-01-02 15:04"), c.diff, c.media.Description, c.media.DateTime, c.media.MediaLink)
		}

		// Show a few entries with zero dates for debugging
		fmt.Printf("\n--- Sample RICH_MEDIA entries with unparseable dates ---\n")
		shown := 0
		for _, item := range data {
			dt := ""
			if v, ok := item["Date/Time"]; ok {
				dt, _ = v.(string)
			}
			parsed := linkedin.ParseFuzzyDateExported(dt)
			if parsed.IsZero() && dt != "" {
				fmt.Printf("  raw: %q\n", dt)
				shown++
				if shown >= 5 {
					break
				}
			}
		}
		if shown == 0 {
			fmt.Println("  (all dates parsed successfully)")
		}
		return
	}

	// Default: print first 3 records as pretty JSON
	limit := 3
	if len(data) < limit {
		limit = len(data)
	}
	for i := 0; i < limit; i++ {
		formatted, _ := json.MarshalIndent(data[i], "", "  ")
		fmt.Printf("\n--- Record %d ---\n%s\n", i, string(formatted))
	}

	if len(data) > limit {
		fmt.Printf("\n... and %d more records (showing first %d)\n", len(data)-limit, limit)
	}
}

func runStatus(client *linkedin.Client) {
	authResp, err := client.CheckAuthorization()
	if err != nil {
		fmt.Printf("Authorization: ⚠ %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Authorization: ✓ active\n")
	var pretty json.RawMessage = authResp
	formatted, _ := json.MarshalIndent(pretty, "               ", "  ")
	fmt.Printf("               %s\n", string(formatted))
}
