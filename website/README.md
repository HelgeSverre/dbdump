# dbdump Website

Simple one-pager marketing site for dbdump with Swiss-IKEA-retro webring aesthetic.

## Design Philosophy

- **Swiss Design**: Clean grid layouts, generous white space, functional typography
- **IKEA Style**: Practical, instructional, step-by-step approach
- **Retro Webring**: Nostalgic 90s/early 2000s web elements, badges, borders

## Features

- Fully responsive
- Dark mode support
- No JavaScript required
- Fast and lightweight
- Grid-based subtle background pattern
- Clear documentation and usage examples

## Local Development

### Option 1: Python (macOS/Linux)

```bash
cd website
python3 -m http.server 8000
```

Open http://localhost:8000

### Option 2: PHP

```bash
cd website
php -S localhost:8000
```

Open http://localhost:8000

### Option 3: Node.js (npx)

```bash
cd website
npx serve
```

### Option 4: Go

```bash
cd website
go run -m http.FileServer http.Dir(".")
```

Or create a simple server:

```bash
cd website
cat > server.go << 'EOF'
package main

import (
    "log"
    "net/http"
)

func main() {
    fs := http.FileServer(http.Dir("."))
    http.Handle("/", fs)
    log.Println("Server starting on http://localhost:8000")
    log.Fatal(http.ListenAndServe(":8000", nil))
}
EOF

go run server.go
```

## Deployment

### GitHub Pages

1. Enable GitHub Pages in repository settings
2. Set source to `main` branch, `/website` folder
3. Site will be available at `https://username.github.io/dbdump/`

### Netlify

1. Create `netlify.toml` in repo root:

```toml
[build]
  publish = "website"
```

2. Connect repository to Netlify
3. Deploy

### Cloudflare Pages

1. Connect repository
2. Set build output directory to `website`
3. Deploy

## File Structure

```
website/
├── index.html      # Main page
├── style.css       # All styles
└── README.md       # This file
```

## Color Scheme

### Light Mode
- Background: `#f5f5f5`
- Text: `#1a1a1a`
- Accent: `#0066cc`
- Yellow: `#ffcc00`
- Blue: `#0051a5`

### Dark Mode
- Background: `#1a1a1a`
- Text: `#f5f5f5`
- Card Background: `#2a2a2a`

## Typography

- Sans-serif: System font stack (SF Pro, Segoe UI, etc.)
- Monospace: SF Mono, Monaco, Cascadia Code, etc.

## Browser Support

- Modern browsers (Chrome, Firefox, Safari, Edge)
- Dark mode via `prefers-color-scheme`
- Graceful degradation for older browsers
