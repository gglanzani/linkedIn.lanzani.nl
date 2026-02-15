package hugo

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var reImageRef = regexp.MustCompile(`!\[.*?\]\(/media/([^)]+)\)`)

// imageSrcLine matches image_sources = ["url1", "url2"] in TOML frontmatter.
var reImageSrc = regexp.MustCompile(`image_sources\s*=\s*\[([^\]]+)\]`)

// DownloadMissingImages scans all markdown files in contentDir, finds image
// references, and downloads any missing images from the source URLs stored
// in the frontmatter's image_sources param.
func DownloadMissingImages(contentDir, mediaDir string) error {
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return fmt.Errorf("reading content dir: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	downloaded := 0
	skipped := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(contentDir, entry.Name())
		imageRefs, sourceURLs, err := parseImageInfo(path)
		if err != nil {
			log.Printf("  ⚠ Error parsing %s: %v", entry.Name(), err)
			continue
		}

		if len(imageRefs) == 0 {
			continue
		}

		for i, ref := range imageRefs {
			destPath := filepath.Join(mediaDir, ref)

			// Already exists?
			if _, err := os.Stat(destPath); err == nil {
				skipped++
				continue
			}

			// Need a source URL
			if i >= len(sourceURLs) {
				log.Printf("  ⚠ No source URL for %s (image %d)", ref, i)
				continue
			}

			srcURL := sourceURLs[i]
			if srcURL == "" {
				log.Printf("  ⚠ Empty source URL for %s", ref)
				continue
			}

			if err := downloadFile(client, srcURL, destPath); err != nil {
				log.Printf("  ⚠ Error downloading %s: %v", ref, err)
				continue
			}

			downloaded++
			log.Printf("  ✓ Downloaded %s", ref)
		}
	}

	log.Printf("Images: %d downloaded, %d already present", downloaded, skipped)
	return nil
}

// parseImageInfo extracts image filenames referenced in the post body and
// source URLs from frontmatter.
func parseImageInfo(path string) (imageRefs []string, sourceURLs []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	text := content.String()

	// Extract image references from body
	for _, match := range reImageRef.FindAllStringSubmatch(text, -1) {
		imageRefs = append(imageRefs, match[1])
	}

	// Extract source URLs from frontmatter
	if m := reImageSrc.FindStringSubmatch(text); len(m) > 1 {
		// Parse the TOML array value: "url1", "url2"
		raw := m[1]
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, `"`)
			if part != "" {
				sourceURLs = append(sourceURLs, part)
			}
		}
	}

	return imageRefs, sourceURLs, nil
}

func downloadFile(client *http.Client, url, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
