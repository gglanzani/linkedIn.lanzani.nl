# linkedin-posts-fetcher

Fetches your LinkedIn posts via the [Member Data Portability APIs](https://learn.microsoft.com/en-us/linkedin/dma/member-data-portability/) and generates Hugo-compatible markdown files.

## Setup

Requires Go 1.25+ and a LinkedIn OAuth token with the `r_dma_portability_member` scope.

```bash
go build -o linkedin-posts-fetcher .
export LINKEDIN_ACCESS_TOKEN="your-token-here"
```

## Commands

### `generate`

Fetch posts from LinkedIn and write Hugo markdown files to the content directory.

```bash
linkedin-posts-fetcher generate
```

On the first run, this does a full fetch via the Snapshot API (MEMBER_SHARE_INFO, ARTICLES, and RICH_MEDIA for images). On subsequent runs within 28 days, it uses the Changelog API for incremental updates, only fetching new posts since the last run. The last fetch timestamp is stored in `.last-fetch` inside the content directory.

Existing files are never overwritten.

### `images`

Download missing images referenced in the generated markdown files.

```bash
linkedin-posts-fetcher images
```

### `status`

Check that your OAuth token is valid and changelog generation is active.

```bash
linkedin-posts-fetcher status
```

### `dump`

Print raw JSON from a snapshot domain for debugging.

```bash
linkedin-posts-fetcher dump                          # first 3 MEMBER_SHARE_INFO records
linkedin-posts-fetcher dump RICH_MEDIA               # first 3 RICH_MEDIA records
linkedin-posts-fetcher dump RICH_MEDIA 2024-06-29    # RICH_MEDIA entries near a date
```

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `LINKEDIN_ACCESS_TOKEN` | Yes (for `generate`, `status`, `dump`) | - | OAuth token with `r_dma_portability_member` scope |
| `CONTENT_DIR` | No | `../content/posts` (relative to binary) | Hugo content/posts directory |
| `MEDIA_DIR` | No | `../static/media` (relative to binary) | Hugo static/media directory |
