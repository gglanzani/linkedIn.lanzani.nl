# linkedin.lanzani.nl

An archive of Giovanni Lanzani's LinkedIn posts, built with [Hugo](https://gohugo.io/) and served at [linkedin.lanzani.nl](https://linkedin.lanzani.nl).

## Overview

This site collects and displays LinkedIn posts in a clean, browsable format using the custom [hugola](https://github.com/gglanzani/hugola) theme (derived from [PureCSS](https://purecss.io/)).

## Local development

### Prerequisites

- [Hugo](https://gohugo.io/installation/) (extended edition)
- Git

### Getting started

```bash
# Clone the repository with submodules
git clone --recurse-submodules git@github.com:gglanzani/linkedIn.lanzani.nl.git
cd linkedIn.lanzani.nl

# Start the development server
hugo server -D
```

The site will be available at `http://localhost:1313/`.

### Adding a new post

```bash
hugo new posts/my-new-post.md
```

### Add views, likes, comments

```sh
uv run --with playwright python scripts/scrape_metrics.py
```

## Deployment

The site is automatically built and deployed to [GitHub Pages](https://pages.github.com/) on every push to `main` via the GitHub Actions workflow in `.github/workflows/hugo.yml`.

### Custom domain setup

To serve the site on `linkedin.lanzani.nl`, configure a `CNAME` DNS record pointing to `gglanzani.github.io` (or the appropriate GitHub Pages domain).

## Project structure

```
.
├── archetypes/        # Hugo content templates
├── assets/            # Site assets (images, etc.)
├── content/posts/     # LinkedIn posts as Markdown files
├── layouts/partials/  # Custom Hugo partial templates
├── static/media/      # Post images and media
├── themes/hugola/     # Hugo theme (git submodule)
├── hugo.toml          # Hugo configuration
└── .github/workflows/ # GitHub Actions CI/CD
```

## License

Content is authored by Giovanni Lanzani. The hugola theme is licensed under the MIT License.
