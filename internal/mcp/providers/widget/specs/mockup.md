# Widget Design Specification — Mockup Module

## Purpose

Create UI mockups and prototypes that demonstrate layout, spacing, and component hierarchy. These are visual representations, not functional UIs.

## Mockup Style

```html
<style>
  .mockup { background: var(--widget-bg); border: 1px solid var(--widget-border); border-radius: var(--widget-radius); overflow: hidden; font-family: system-ui; }
  .mockup-header { background: var(--widget-muted); padding: 8px 12px; border-bottom: 1px solid var(--widget-border); display: flex; align-items: center; gap: 8px; }
  .mockup-dots { display: flex; gap: 4px; }
  .mockup-dot { width: 8px; height: 8px; border-radius: 50%; }
  .mockup-dot-red { background: #ef4444; }
  .mockup-dot-yellow { background: #f59e0b; }
  .mockup-dot-green { background: #10b981; }
  .mockup-title { font-size: 0.75rem; color: var(--widget-muted-fg); }
  .mockup-body { padding: 16px; }
  .mockup-placeholder { background: var(--widget-muted); border-radius: 4px; height: 16px; margin-bottom: 8px; }
  .mockup-placeholder.w-75 { width: 75%; }
  .mockup-placeholder.w-50 { width: 50%; }
  .mockup-placeholder.w-full { width: 100%; }
  .mockup-placeholder.h-lg { height: 120px; }
</style>
```

## Window Chrome

Always include window chrome for app mockups:
```html
<div class="mockup">
  <div class="mockup-header">
    <div class="mockup-dots">
      <div class="mockup-dot mockup-dot-red"></div>
      <div class="mockup-dot mockup-dot-yellow"></div>
      <div class="mockup-dot mockup-dot-green"></div>
    </div>
    <div class="mockup-title">App Title</div>
  </div>
  <div class="mockup-body">
    <!-- mockup content -->
  </div>
</div>
```

## Placeholder Elements

Use gray boxes to indicate content areas:
- Text lines: height 16px, varying widths
- Images: height 120px, full width
- Buttons: use actual styled buttons (can be interactive via sendPrompt)
- Icons: use emoji or simple SVG shapes

## Annotations

Add notes to explain design decisions:
```html
<div style="display: flex; align-items: flex-start; gap: 8px; margin-top: 8px; padding: 8px; background: var(--widget-accent); border-radius: var(--widget-radius); font-size: 0.75rem; color: var(--widget-muted-fg);">
  💡 Note: Sidebar collapses to icons on mobile
</div>
```

## Interactive Variants

Let users explore different design options:
```html
<button class="action-btn" onclick="sendPrompt('Show dark mode version')">🌙 Dark Mode</button>
<button class="action-btn" onclick="sendPrompt('Show mobile layout')">📱 Mobile</button>
```
