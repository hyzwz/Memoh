# Widget Design Specification — Chart Module

## Chart.js Setup

Load Chart.js from CDN in your `<script>`:
```html
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
  const ctx = document.querySelector('#myChart').getContext('2d');
  new Chart(ctx, { /* config */ });
</script>
```

## Canvas Element

Always provide explicit dimensions via the container:
```html
<div style="position: relative; width: 100%; height: 300px;">
  <canvas id="myChart"></canvas>
</div>
```

## Chart Configuration

```javascript
{
  type: 'bar', // bar, line, pie, doughnut, radar, polarArea
  data: {
    labels: [...],
    datasets: [{
      label: 'Dataset',
      data: [...],
      backgroundColor: [/* use color palette below */],
      borderColor: [/* use color palette below */],
      borderWidth: 1
    }]
  },
  options: {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: true, position: 'bottom' },
      title: { display: false } // use widget title prop instead
    }
  }
}
```

## Color Palette

Use these colors in order for datasets:
```javascript
const colors = [
  '#3b82f6', // blue
  '#ef4444', // red
  '#10b981', // green
  '#f59e0b', // amber
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#06b6d4', // cyan
  '#f97316', // orange
];
```

For backgrounds, use 20% opacity: `rgba(59, 130, 246, 0.2)`

## Chart Types Guide

- **Bar/Line**: Best for time series, comparisons. Use line for trends, bar for categories.
- **Pie/Doughnut**: Best for part-to-whole (max 6-8 slices). Use doughnut over pie.
- **Radar**: Best for multi-dimensional comparison (3-8 axes).
- **Mixed**: Combine bar + line for dual-axis visualizations.

## Interactive Charts

Add click handlers to trigger sendPrompt:
```javascript
myChart.canvas.addEventListener('click', (e) => {
  const points = myChart.getElementsAtEventForMode(e, 'nearest', { intersect: true }, true);
  if (points.length) {
    const { index } = points[0];
    const label = myChart.data.labels[index];
    sendPrompt(`Tell me more about ${label}`);
  }
});
```

## Animations

- Keep animations short: `animation: { duration: 500 }`
- Disable on streaming: check if content is still loading
- Use `animation: false` if the chart updates frequently
