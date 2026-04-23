#!/usr/bin/env python3
"""
Scrape LinkedIn post metrics (likes, comments, views) using Playwright.

Usage:
    # First run: manual login (saves auth state for later)
    python scripts/scrape_metrics.py --login

    # Subsequent runs: headless scrape of all posts
    python scripts/scrape_metrics.py

    # Force re-scrape posts that already have non-zero metrics
    python scripts/scrape_metrics.py --force
"""

import argparse
import random
import re
import sys
import time
from datetime import datetime, timedelta
from pathlib import Path

from playwright.sync_api import TimeoutError as PlaywrightTimeout
from playwright.sync_api import sync_playwright

# --- Configuration ---
POSTS_DIR = Path(__file__).resolve().parent.parent / "content" / "posts"
AUTH_DIR = Path(__file__).resolve().parent / ".auth"
AUTH_STATE_FILE = AUTH_DIR / "state.json"

FRONTMATTER_RE = re.compile(r"^\+\+\+\n(.*?)\n\+\+\+", re.DOTALL)

LIKES_SELECTOR = (
    "p._3a5099c8._1f7b0faa._0fba6839._9500257f._8961e6a3."
    "b17e1ae9._524dc3d4._1e605ce8._549691dd._0447a2ac"
)
COMMENTS_SELECTOR = (
    "p._3a5099c8._1f7b0faa._0fba6839._9500257f._8961e6a3."
    "b17e1ae9._524dc3d4._1e605ce8.d8b6e5f8"
)
IMPRESSIONS_SELECTOR = (
    "p._3a5099c8._1f7b0faa.caa4ef5c._8b08c498._975606c5."
    "ad4ef320._0e7674a8.a644e2c2._4646031b.d8b6e5f8"
)
MAX_POST_AGE = timedelta(days=365)


def parse_frontmatter(content: str) -> tuple[str, str, str]:
    """Split a markdown file into (frontmatter, body, raw_match).

    Returns the frontmatter text (between +++ markers), the body after,
    and the full matched block including markers.
    """
    m = FRONTMATTER_RE.match(content)
    if not m:
        return "", content, ""
    fm_text = m.group(1)
    body = content[m.end() :]
    return fm_text, body, m.group(0)


def extract_url(fm_text: str) -> str | None:
    """Extract the url param from TOML frontmatter."""
    m = re.search(r'^\s*url\s*=\s*"([^"]+)"', fm_text, re.MULTILINE)
    return m.group(1) if m else None


def extract_metric(fm_text: str, key: str) -> int | None:
    """Extract an integer metric from frontmatter, e.g. likes = 5."""
    m = re.search(rf"^\s*{key}\s*=\s*(\d+)", fm_text, re.MULTILINE)
    return int(m.group(1)) if m else None


def extract_post_date(fm_text: str, md_file: Path) -> datetime | None:
    """Extract the post date from frontmatter, falling back to filename."""
    m = re.search(r'^\s*date\s*=\s*"([^"]+)"', fm_text, re.MULTILINE)
    if m:
        try:
            return datetime.fromisoformat(m.group(1))
        except ValueError:
            pass

    m = re.match(r"(\d{4}-\d{2}-\d{2})-", md_file.name)
    if m:
        try:
            return datetime.strptime(m.group(1), "%Y-%m-%d")
        except ValueError:
            pass

    return None


def has_nonzero_metrics(fm_text: str) -> bool:
    """Check if the post already has non-zero likes, views, or comments."""
    likes = extract_metric(fm_text, "likes") or 0
    views = extract_metric(fm_text, "views") or 0
    comments = extract_metric(fm_text, "comments") or 0
    return (likes + views + comments) > 0


def update_frontmatter(fm_text: str, likes: int, views: int, comments: int) -> str:
    """Update likes, views and comments in the frontmatter text.

    - Replaces existing likes = ... and views = ... lines.
    - Adds comments = ... after the views line if not present, or updates it.
    """
    # Update likes
    fm_text = re.sub(
        r"^(\s*likes\s*=\s*)\d+",
        rf"\g<1>{likes}",
        fm_text,
        flags=re.MULTILINE,
    )

    # Update views
    fm_text = re.sub(
        r"^(\s*views\s*=\s*)\d+",
        rf"\g<1>{views}",
        fm_text,
        flags=re.MULTILINE,
    )

    # Update or add comments
    if re.search(r"^\s*comments\s*=\s*\d+", fm_text, re.MULTILINE):
        fm_text = re.sub(
            r"^(\s*comments\s*=\s*)\d+",
            rf"\g<1>{comments}",
            fm_text,
            flags=re.MULTILINE,
        )
    else:
        # Insert comments after the views line
        fm_text = re.sub(
            r"^(\s*views\s*=\s*\d+)",
            rf"\1\n  comments = {comments}",
            fm_text,
            flags=re.MULTILINE,
        )

    return fm_text


def parse_metric_text(text: str, *, suffix: str | None = None) -> int:
    """Extract an integer metric from element text."""
    cleaned = " ".join(text.strip().split())
    if not cleaned:
        return 0

    if suffix:
        m = re.search(rf"([\d,]+)\s+{re.escape(suffix)}\b", cleaned, re.IGNORECASE)
        if m:
            return int(m.group(1).replace(",", ""))

    m = re.search(r"([\d,]+)", cleaned)
    if m:
        return int(m.group(1).replace(",", ""))

    return 0


def extract_metric_from_selector(page, selector: str, *, suffix: str | None = None) -> int:
    """Return the first metric found for a selector, or 0 if none is found."""
    for el in page.query_selector_all(selector):
        value = parse_metric_text(el.inner_text(), suffix=suffix)
        if value:
            return value
    return 0


def scrape_post(page, url: str) -> dict:
    """Visit a LinkedIn post URL and scrape metrics.

    Returns dict with keys: likes, comments, views (all int, default 0).
    """
    metrics = {"likes": 0, "comments": 0, "views": 0}

    try:
        page.goto(url, wait_until="domcontentloaded", timeout=30000)
        # Wait a moment for dynamic content to load
        page.wait_for_timeout(3000)
    except PlaywrightTimeout:
        print(f"    [WARN] Timeout loading {url}")
        return metrics

    # --- Likes ---
    try:
        metrics["likes"] = extract_metric_from_selector(page, LIKES_SELECTOR)
    except Exception as e:
        print(f"    [WARN] Error scraping likes: {e}")

    # --- Comments ---
    try:
        metrics["comments"] = extract_metric_from_selector(page, COMMENTS_SELECTOR)
    except Exception as e:
        print(f"    [WARN] Error scraping comments: {e}")

    # --- Views / Impressions ---
    try:
        metrics["views"] = extract_metric_from_selector(
            page, IMPRESSIONS_SELECTOR, suffix="impressions"
        )
    except Exception as e:
        print(f"    [WARN] Error scraping views: {e}")

    return metrics


def do_login():
    """Open a visible browser for the user to log into LinkedIn manually."""
    AUTH_DIR.mkdir(parents=True, exist_ok=True)

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        context = browser.new_context()
        page = context.new_page()
        page.goto("https://www.linkedin.com/login")

        print()
        print("=" * 60)
        print(" Please log into LinkedIn in the browser window.")
        print(" After you're logged in and see your feed,")
        print(" come back here and press ENTER to save the session.")
        print("=" * 60)
        print()

        input("Press ENTER when you're logged in... ")

        # Save auth state
        context.storage_state(path=str(AUTH_STATE_FILE))
        print(f"Auth state saved to {AUTH_STATE_FILE}")

        browser.close()


def do_scrape(force: bool = False):
    """Scrape metrics for all posts."""
    if not AUTH_STATE_FILE.exists():
        print(
            "ERROR: No auth state found. Run with --login first to log into LinkedIn."
        )
        sys.exit(1)

    # Collect all markdown files
    md_files = sorted(POSTS_DIR.glob("*.md"), reverse=True)
    print(f"Found {len(md_files)} posts in {POSTS_DIR}")
    cutoff = datetime.now() - MAX_POST_AGE

    # Pre-filter: collect posts that need scraping
    to_scrape = []
    for md_file in md_files:
        content = md_file.read_text(encoding="utf-8")
        fm_text, body, raw = parse_frontmatter(content)
        if not fm_text:
            continue
        post_date = extract_post_date(fm_text, md_file)
        if post_date and post_date < cutoff:
            continue
        url = extract_url(fm_text)
        if not url:
            continue
        if not force and has_nonzero_metrics(fm_text):
            continue
        to_scrape.append((md_file, content, fm_text, body, url))

    print(
        f"Posts to scrape: {len(to_scrape)} (skipped {len(md_files) - len(to_scrape)})"
    )
    if not to_scrape:
        print("Nothing to scrape. Use --force to re-scrape all posts.")
        return

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(storage_state=str(AUTH_STATE_FILE))
        page = context.new_page()

        for i, (md_file, content, fm_text, body, url) in enumerate(to_scrape, 1):
            print(f"[{i}/{len(to_scrape)}] {md_file.name}")
            print(f"    URL: {url}")

            metrics = scrape_post(page, url)
            print(
                f"    Likes: {metrics['likes']}, "
                f"Comments: {metrics['comments']}, "
                f"Views: {metrics['views']}"
            )

            # Update frontmatter and write back
            new_fm = update_frontmatter(
                fm_text, metrics["likes"], metrics["views"], metrics["comments"]
            )
            new_content = f"+++\n{new_fm}\n+++{body}"
            md_file.write_text(new_content, encoding="utf-8")

            # Rate limit: 2-3 seconds between requests
            if i < len(to_scrape):
                delay = 2 + random.random()
                time.sleep(delay)

        browser.close()

    print()
    print("Done! All posts updated.")


def main():
    parser = argparse.ArgumentParser(
        description="Scrape LinkedIn post metrics using Playwright"
    )
    parser.add_argument(
        "--login",
        action="store_true",
        help="Open a browser for manual LinkedIn login (saves auth state)",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Re-scrape posts that already have non-zero metrics",
    )
    args = parser.parse_args()

    if args.login:
        do_login()
    else:
        do_scrape(force=args.force)


if __name__ == "__main__":
    main()
