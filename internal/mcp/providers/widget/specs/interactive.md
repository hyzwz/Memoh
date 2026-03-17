# Widget Design Specification — Interactive Module

## Component Types

### Metric Card
Display key statistics with optional trend indicator:
```html
<style>
  .metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; }
  .metric-card { background: var(--widget-muted); border-radius: var(--widget-radius); padding: 16px; }
  .metric-value { font-size: 1.75rem; font-weight: 700; color: var(--widget-fg); }
  .metric-label { font-size: 0.75rem; color: var(--widget-muted-fg); margin-top: 4px; }
  .metric-trend { font-size: 0.75rem; margin-top: 4px; }
  .trend-up { color: #10b981; }
  .trend-down { color: #ef4444; }
</style>
<div class="metric-grid">
  <div class="metric-card">
    <div class="metric-value">1,234</div>
    <div class="metric-label">Total Users</div>
    <div class="metric-trend trend-up">↑ 12% vs last month</div>
  </div>
</div>
```

### Action Button List
Provide choices that trigger AI follow-up:
```html
<style>
  .action-list { display: flex; flex-direction: column; gap: 8px; }
  .action-btn {
    display: flex; align-items: center; gap: 8px;
    padding: 10px 14px; border-radius: var(--widget-radius);
    border: 1px solid var(--widget-border); background: var(--widget-bg);
    color: var(--widget-fg); cursor: pointer; font-size: 0.875rem;
    transition: background 0.15s;
  }
  .action-btn:hover { background: var(--widget-muted); }
</style>
<div class="action-list">
  <button class="action-btn" onclick="sendPrompt('Analyze sales data')">📊 Analyze Sales</button>
  <button class="action-btn" onclick="sendPrompt('Generate report')">📄 Generate Report</button>
</div>
```

### Data Table
For tabular data display:
```html
<style>
  .data-table { width: 100%; border-collapse: collapse; font-size: 0.8125rem; }
  .data-table th { text-align: left; padding: 8px; color: var(--widget-muted-fg); border-bottom: 1px solid var(--widget-border); font-weight: 500; }
  .data-table td { padding: 8px; border-bottom: 1px solid var(--widget-border); color: var(--widget-fg); }
  .data-table tr:hover td { background: var(--widget-muted); }
</style>
<table class="data-table">
  <thead><tr><th>Name</th><th>Value</th><th>Action</th></tr></thead>
  <tbody>
    <tr>
      <td>Item A</td><td>100</td>
      <td><button class="action-btn" onclick="sendPrompt('Details about Item A')">View</button></td>
    </tr>
  </tbody>
</table>
```

### Form / Input
Collect structured input from users:
```html
<style>
  .widget-form { display: flex; flex-direction: column; gap: 12px; }
  .form-group label { font-size: 0.75rem; color: var(--widget-muted-fg); display: block; margin-bottom: 4px; }
  .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--widget-border); border-radius: var(--widget-radius); background: var(--widget-bg); color: var(--widget-fg); font-size: 0.875rem; }
  .form-submit { padding: 8px 16px; background: var(--widget-primary); color: var(--widget-primary-fg); border: none; border-radius: var(--widget-radius); cursor: pointer; font-size: 0.875rem; }
</style>
```

## Layout Rules

- Max width: 100% of container (no overflow)
- Padding: 16px inside widget boundary
- Font sizes: values 1.75rem, labels 0.75rem, body 0.875rem
- Use `gap` for spacing, not margins on children
- All interactive elements must have `cursor: pointer`
- Hover states required for clickable elements
