# Widget Design Specification — Chart Module

## CRITICAL: Chart Container Sizing

**ALL chart containers MUST have explicit inline width AND height styles.** Without this, the chart will NOT render (the container collapses to 0px height).

```html
<!-- CORRECT — always include inline style with explicit height -->
<div id="chart" style="width: 100%; height: 400px;"></div>

<!-- WRONG — chart will NOT render (height is 0) -->
<div id="chart"></div>
<div id="chart" class="chart-container"></div>
```

This applies to BOTH Chart.js canvas containers AND ECharts div containers. CSS classes alone are NOT sufficient — use inline `style="width: 100%; height: XXXpx;"` on every chart element.

## Library Selection Guide

| Need | Use | Why |
|------|-----|-----|
| Pie, bar, line, radar, doughnut | **Chart.js** | Simple config, lightweight |
| Gantt, heatmap, treemap, sankey, funnel, gauge, sunburst, map | **ECharts** | 30+ built-in chart types |
| Highly custom SVG | **D3.js** | Full SVG control |

---

## Chart.js (Simple Charts)

### Setup
```html
<div style="position: relative; width: 100%; height: 300px;">
  <canvas id="myChart"></canvas>
</div>
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
  const ctx = document.querySelector('#myChart').getContext('2d');
  new Chart(ctx, { /* config */ });
</script>
```

### Configuration
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

### Chart Types
- **Bar/Line**: Time series, comparisons. Line for trends, bar for categories.
- **Pie/Doughnut**: Part-to-whole (max 6-8 slices). Prefer doughnut.
- **Radar**: Multi-dimensional comparison (3-8 axes).
- **Mixed**: Combine bar + line for dual-axis.

### Interactive Click
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

---

## ECharts (Advanced Charts)

### Setup
```html
<div id="chart" style="width: 100%; height: 400px;"></div>
<script src="https://cdn.jsdelivr.net/npm/echarts/dist/echarts.min.js"></script>
<script>
  var chart = echarts.init(document.getElementById('chart'));
  chart.setOption({ /* config */ });
  window.addEventListener('resize', () => chart.resize());
</script>
```

**IMPORTANT**: The `<div>` MUST have `style="width: 100%; height: 400px;"` as an inline style attribute. Without explicit height, ECharts will NOT render (container collapses to 0px). Do NOT rely on CSS classes for sizing. Always call `chart.resize()` on window resize for responsive behavior.

### Gantt Chart (ECharts)

Use ECharts `custom` series type to render Gantt charts:

```javascript
// Data: each task has [categoryIndex, startTimestamp, endTimestamp, taskName]
var tasks = [
  { name: '需求分析', start: '2026-01-01', end: '2026-01-31', category: '规划', progress: 100 },
  { name: '系统设计', start: '2026-02-01', end: '2026-02-28', category: '规划', progress: 100 },
  { name: '前端开发', start: '2026-03-01', end: '2026-05-31', category: '开发', progress: 65 },
  { name: '后端开发', start: '2026-03-15', end: '2026-06-15', category: '开发', progress: 55 },
  { name: '测试', start: '2026-06-01', end: '2026-07-31', category: '测试', progress: 0 },
];

var categories = tasks.map(t => t.name);
var categoryColors = { '规划': '#3b82f6', '开发': '#10b981', '测试': '#f59e0b' };

chart.setOption({
  tooltip: {
    formatter: function(p) {
      var t = tasks[p.dataIndex];
      return t.name + '<br/>' + t.start + ' → ' + t.end + '<br/>进度: ' + t.progress + '%';
    }
  },
  grid: { left: 120, right: 30, top: 30, bottom: 30 },
  xAxis: {
    type: 'time',
    position: 'top',
    axisLabel: { formatter: '{MM}/{dd}' },
    splitLine: { show: true, lineStyle: { type: 'dashed', color: '#333' } }
  },
  yAxis: {
    type: 'category',
    data: categories,
    inverse: true,
    axisLabel: { width: 100, overflow: 'truncate' }
  },
  series: [{
    type: 'custom',
    renderItem: function(params, api) {
      var catIndex = api.value(0);
      var start = api.coord([api.value(1), catIndex]);
      var end = api.coord([api.value(2), catIndex]);
      var height = api.size([0, 1])[1] * 0.6;
      var width = end[0] - start[0];
      var progress = api.value(3);
      return {
        type: 'group',
        children: [
          // Background bar
          { type: 'rect', shape: { x: start[0], y: start[1] - height/2, width: width, height: height, r: 4 },
            style: { fill: api.visual('color'), opacity: 0.3 } },
          // Progress bar
          { type: 'rect', shape: { x: start[0], y: start[1] - height/2, width: width * progress / 100, height: height, r: 4 },
            style: { fill: api.visual('color') } },
          // Label
          width > 50 ? { type: 'text', style: { text: progress + '%', x: start[0] + width/2, y: start[1],
            fill: '#fff', fontSize: 11, align: 'center', verticalAlign: 'middle' } } : null
        ].filter(Boolean)
      };
    },
    encode: { x: [1, 2], y: 0 },
    data: tasks.map(function(t, i) {
      return {
        value: [i, new Date(t.start).getTime(), new Date(t.end).getTime(), t.progress],
        itemStyle: { color: categoryColors[t.category] || '#3b82f6' }
      };
    })
  },
  // Today marker line
  {
    type: 'line', markLine: { silent: true, symbol: 'none',
      lineStyle: { color: '#ef4444', type: 'solid', width: 2 },
      data: [{ xAxis: new Date().getTime() }],
      label: { formatter: '今天', position: 'start' }
    }, data: []
  }]
});

chart.on('click', function(params) {
  if (params.dataIndex !== undefined) {
    sendPrompt('详细说明任务: ' + tasks[params.dataIndex].name);
  }
});
```

### Heatmap (ECharts)
```javascript
chart.setOption({
  tooltip: { position: 'top' },
  xAxis: { type: 'category', data: ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'] },
  yAxis: { type: 'category', data: ['Morning','Afternoon','Evening'] },
  visualMap: { min: 0, max: 100, calculable: true, orient: 'horizontal', left: 'center', bottom: 0 },
  series: [{ type: 'heatmap', data: [[0,0,50],[0,1,80],[0,2,30]/* ... */], label: { show: true } }]
});
```

### Other ECharts Types
- **Treemap**: `type: 'treemap'` — hierarchical data visualization
- **Sankey**: `type: 'sankey'` — flow diagrams
- **Funnel**: `type: 'funnel'` — conversion funnels
- **Gauge**: `type: 'gauge'` — dashboard meters
- **Sunburst**: `type: 'sunburst'` — multi-level pie chart

Refer to ECharts documentation for each type's configuration.

### ECharts Dark Theme
For dark backgrounds, set `echarts.init(el, 'dark')` or configure:
```javascript
chart.setOption({
  backgroundColor: 'transparent',
  textStyle: { color: '#e5e5e5' },
  // ... axis and other label colors
});
```

---

## Color Palette

Use these colors in order for datasets (works with both Chart.js and ECharts):
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

---

## Animations

- Keep animations short: Chart.js `animation: { duration: 500 }`, ECharts `animation: true, animationDuration: 500`
- Disable on streaming: check if content is still loading
- Use `animation: false` if the chart updates frequently
