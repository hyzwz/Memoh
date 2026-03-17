# Widget Design Specification — Art Module

## Purpose

Create decorative illustrations, icons, and visual elements using inline SVG. These are aesthetic additions to enhance the chat experience.

## SVG Art Rules

- Use inline SVG only (no external images)
- Keep SVG simple: < 5KB recommended, < 20KB maximum
- Use theme-aware colors from CSS variables
- Set explicit viewBox for consistent scaling

## Color Usage

```css
.art-primary { fill: var(--widget-primary); }
.art-accent { fill: var(--widget-accent); }
.art-muted { fill: var(--widget-muted-fg); opacity: 0.5; }
```

Accent colors for illustrations:
```
#3b82f6 (blue), #10b981 (green), #f59e0b (amber),
#8b5cf6 (violet), #ec4899 (pink), #06b6d4 (cyan)
```

## Animation

Simple CSS animations are allowed for art:
```css
@keyframes float { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-4px); } }
.floating { animation: float 3s ease-in-out infinite; }
```

Rules:
- Max 2 animations per widget
- Duration: 2s–5s (subtle, not distracting)
- Use `prefers-reduced-motion` media query
- No JavaScript-driven animations

## Icon Sets

For simple icons, use SVG paths:
```html
<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
  <circle cx="12" cy="12" r="10" />
  <path d="M8 14s1.5 2 4 2 4-2 4-2" />
  <line x1="9" y1="9" x2="9.01" y2="9" />
  <line x1="15" y1="9" x2="15.01" y2="9" />
</svg>
```

## Composition

- Center art within the widget
- Add padding: minimum 16px around SVG
- Use `max-width: 300px` for standalone illustrations
- Combine with text for context
