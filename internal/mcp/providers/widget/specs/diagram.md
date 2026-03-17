# Widget Design Specification — Diagram Module

## SVG Diagrams

Use inline SVG for flowcharts, trees, and simple graphs. Do NOT use external libraries for simple diagrams.

## Flowchart Pattern

```html
<style>
  .diagram-container { width: 100%; overflow-x: auto; }
  .diagram-container svg { max-width: 100%; height: auto; }
  .node rect { fill: var(--widget-muted); stroke: var(--widget-border); stroke-width: 1; rx: 6; }
  .node text { fill: var(--widget-fg); font-size: 12px; font-family: system-ui; }
  .edge line, .edge path { stroke: var(--widget-muted-fg); stroke-width: 1.5; fill: none; }
  .edge polygon { fill: var(--widget-muted-fg); }
  .node-clickable { cursor: pointer; }
  .node-clickable:hover rect { fill: var(--widget-accent); }
</style>
<div class="diagram-container">
  <svg viewBox="0 0 400 200" xmlns="http://www.w3.org/2000/svg">
    <!-- nodes and edges -->
  </svg>
</div>
```

## Node Sizing

- Standard node: 120x40px
- Decision node (diamond): 80x80px
- Text padding: 8px horizontal, 6px vertical
- Edge gap between nodes: 40px minimum

## Arrow Markers

```svg
<defs>
  <marker id="arrow" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
    <polygon points="0 0, 10 3.5, 0 7" fill="var(--widget-muted-fg)" />
  </marker>
</defs>
```

## Interactive Nodes

Make nodes clickable with sendPrompt:
```svg
<g class="node node-clickable" onclick="sendPrompt('Explain the authentication step')">
  <rect x="10" y="10" width="120" height="40" />
  <text x="70" y="35" text-anchor="middle">Auth</text>
</g>
```

## Complex Diagrams (D3.js)

For graphs with > 10 nodes, use D3.js:
```html
<script src="https://cdn.jsdelivr.net/npm/d3"></script>
<script>
  const data = { nodes: [...], links: [...] };
  // D3 force layout
</script>
```

## Layout Direction

- Default: left-to-right (LR) for process flows
- Top-to-bottom (TB) for hierarchies
- Maintain consistent direction within one diagram
