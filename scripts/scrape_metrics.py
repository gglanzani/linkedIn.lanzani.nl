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
from pathlib import Path

from playwright.sync_api import TimeoutError as PlaywrightTimeout
from playwright.sync_api import sync_playwright

# --- Configuration ---
POSTS_DIR = Path(__file__).resolve().parent.parent / "content" / "posts"
AUTH_DIR = Path(__file__).resolve().parent / ".auth"
AUTH_STATE_FILE = AUTH_DIR / "state.json"

FRONTMATTER_RE = re.compile(r"^\+\+\+\n(.*?)\n\+\+\+", re.DOTALL)


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
        likes_el = page.query_selector(
            "span.social-details-social-counts__social-proof-text"
        )
        likes_fallback = page.query_selector(
            "span.social-details-social-counts__reactions-count"
        )
        if likes_el:
            text = likes_el.inner_text().strip()
            # Formats: "N others" or just a number, or "Name and N others"
            m = re.search(r"([\d,]+)\s+others?", text)
            if m:
                metrics["likes"] = int(m.group(1).replace(",", "")) + 1
            else:
                # Try plain number
                m = re.search(r"^([\d,]+)$", text)
                if m:
                    metrics["likes"] = int(m.group(1).replace(",", ""))
                else:
                    # Single person liked (no "others"), so it's 1
                    metrics["likes"] = 1
        elif likes_fallback:
            metrics["likes"] = likes_fallback.inner_text().strip()
    except Exception as e:
        print(f"    [WARN] Error scraping likes: {e}")

    # --- Comments ---
    try:
        comment_buttons = page.query_selector_all(
            "button.social-details-social-counts__count-value-button"
        )
        for btn in comment_buttons:
            text = btn.inner_text().strip().lower()
            m = re.search(r"([\d,]+)\s*comment", text)
            if m:
                metrics["comments"] = int(m.group(1).replace(",", ""))
                break
        # Alternative selector
        if metrics["comments"] == 0:
            comment_spans = page.query_selector_all(
                "span.social-details-social-counts__count-value"
            )
            for span in comment_spans:
                text = span.inner_text().strip().lower()
                m = re.search(r"([\d,]+)\s*comment", text)
                if m:
                    metrics["comments"] = int(m.group(1).replace(",", ""))
                    break
    except Exception as e:
        print(f"    [WARN] Error scraping comments: {e}")

    # --- Views / Impressions ---
    try:
        views_el = page.query_selector("span.ca-entry-point__num-views strong")
        if views_el:
            text = views_el.inner_text().strip()
            m = re.search(r"([\d,]+)", text)
            if m:
                metrics["views"] = int(m.group(1).replace(",", ""))
        # Alternative: try the analytics-like selector
        if metrics["views"] == 0:
            views_el2 = page.query_selector("span.ca-entry-point__num-views")
            if views_el2:
                text = views_el2.inner_text().strip()
                m = re.search(r"([\d,]+)", text)
                if m:
                    metrics["views"] = int(m.group(1).replace(",", ""))
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

    # Pre-filter: collect posts that need scraping
    to_scrape = []
    for md_file in md_files:
        content = md_file.read_text(encoding="utf-8")
        fm_text, body, raw = parse_frontmatter(content)
        if not fm_text:
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
