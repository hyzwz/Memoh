# Widget Design Specification — Shared Rules

## Environment

Your HTML runs inside a Shadow DOM attached to the chat message. This means:
- Your styles are isolated from the host page
- You have access to CSS custom properties from the host theme
- The only JavaScript API exposed to you is `sendPrompt(text)`

## Theme Variables

Use these CSS variables for theme-aware styling:

```css
:host {
  --widget-bg: var(--background, #ffffff);
  --widget-fg: var(--foreground, #0a0a0a);
  --widget-muted: var(--muted, #f5f5f5);
  --widget-muted-fg: var(--muted-foreground, #737373);
  --widget-border: var(--border, #e5e5e5);
  --widget-primary: var(--primary, #171717);
  --widget-primary-fg: var(--primary-foreground, #fafafa);
  --widget-accent: var(--accent, #f5f5f5);
  --widget-radius: var(--radius, 0.5rem);
}
```

## Structure

Always output in this order:
1. `<style>` block (theme-aware, use CSS variables)
2. HTML content
3. `<script>` block (optional, for Chart.js/D3 initialization)

## Interactivity

The ONLY way to interact with the AI is via `sendPrompt(text)`:
```html
<button onclick="sendPrompt('Show me more details about Q2')">View Details</button>
```

Rules:
- `sendPrompt` accepts a string up to 500 characters
- Rate limited to 5 calls per minute
- Use descriptive prompts that the AI can act on

## CDN Allowlist

Only these CDN sources are permitted:
- `https://cdn.jsdelivr.net/npm/chart.js` — Chart.js (pie, bar, line, radar, doughnut, polarArea)
- `https://cdn.jsdelivr.net/npm/echarts/dist/echarts.min.js` — ECharts (Gantt, heatmap, treemap, sankey, gauge, map, funnel, sunburst, and 30+ chart types)
- `https://cdn.jsdelivr.net/npm/d3` — D3.js (custom SVG visualizations)

### When to use which library
- **Chart.js**: Simple charts (pie, bar, line, radar). Lightweight, easy config.
- **ECharts**: Complex/advanced visualizations (Gantt, heatmap, treemap, sankey, funnel, gauge, geographic maps, parallel coordinates, sunburst). Use ECharts when Chart.js doesn't support the chart type.
- **D3.js**: Only for highly custom SVG layouts that neither Chart.js nor ECharts can handle.

## Security

- NO `<iframe>`, `<object>`, `<embed>` tags
- NO external stylesheets (`<link rel="stylesheet">`)
- NO dynamic code generation or string-to-code conversion
- NO inline event handlers except `onclick="sendPrompt('...')"`

## Responsive Design

- Use percentage widths or `max-width` instead of fixed pixel widths
- Charts should use `responsive: true` and `maintainAspectRatio: false`
- Test your layout at 300px–800px width range

## Streaming Compatibility

Your HTML will be rendered incrementally via morphdom as tokens arrive. To ensure visual stability:
- Put `<style>` first so styles apply before content renders
- Only use DOM manipulation methods (createElement, appendChild) — never inject HTML via string methods
- Chart.js / ECharts initialization should be in a `<script>` at the end
- Use `requestAnimationFrame` or `setTimeout(fn, 0)` for DOM-dependent initialization
