# UI Design System Migration (Phase 1) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate OpenClaw Office's visual design system into Memoh — glass morphism, glow effects, animations, Deep Space theme, and cyan+purple dual-tone color scheme.

**Architecture:** Extend existing CSS variable system in `style.css` with OpenClaw's design tokens. Add glass/glow/animation utility classes. Extend Pinia settings store to support 3 themes (light/dark/deep-space). Update settings UI for theme switcher.

**Tech Stack:** Tailwind CSS v4, Vue 3, Pinia, VueUse useColorMode, CSS custom properties

---

### Task 1: Add OpenClaw Design Tokens to style.css

**Files:**
- Modify: `apps/web/src/style.css`

**Step 1: Add new CSS variables to `:root` (light mode)**

Add after the existing `:root` block's `--sidebar-ring` line:

```css
  /* === OpenClaw Design Tokens === */
  /* Text hierarchy */
  --text-primary:   #1e293b;
  --text-secondary: #64748b;
  --text-muted:     #94a3b8;
  --text-accent:    #0891b2;

  /* Accent colors (cyan + purple dual-tone) */
  --accent-primary:   #0891b2;
  --accent-secondary: #7c3aed;
  --accent-success:   #059669;
  --accent-warning:   #d97706;
  --accent-danger:    #dc2626;
  --accent-glow:      rgba(8, 145, 178, 0.1);

  /* Glass effect surfaces */
  --glass-light:  rgba(0, 0, 0, 0.02);
  --glass-medium: rgba(0, 0, 0, 0.05);
  --glass-heavy:  rgba(0, 0, 0, 0.08);
  --glass-border: rgba(0, 0, 0, 0.08);

  /* Surface hierarchy */
  --surface-base:     #f8fafc;
  --surface-card:     #ffffff;
  --surface-elevated: #ffffff;

  /* Office zone colors */
  --zone-desk: #e8edf5;
  --zone-meeting: #e0eaf5;
  --zone-hotDesk: #e5e8f0;
  --zone-lounge: #e8e5f0;
  --zone-stroke: #c8d0dc;
  --zone-label: #64748b;
```

**Step 2: Add dark mode overrides to `.dark` block**

Add after the existing `.dark` block's `--sidebar-ring` line:

```css
  /* === OpenClaw Design Tokens (dark) === */
  --text-primary:   #f1f5f9;
  --text-secondary: #94a3b8;
  --text-muted:     #64748b;
  --text-accent:    #06b6d4;

  --accent-primary:   #06b6d4;
  --accent-secondary: #8b5cf6;
  --accent-success:   #10b981;
  --accent-warning:   #f59e0b;
  --accent-danger:    #ef4444;
  --accent-glow:      rgba(6, 182, 212, 0.2);

  --glass-light:  rgba(255, 255, 255, 0.03);
  --glass-medium: rgba(255, 255, 255, 0.06);
  --glass-heavy:  rgba(255, 255, 255, 0.10);
  --glass-border: rgba(255, 255, 255, 0.06);

  --surface-base:     #0f172a;
  --surface-card:     #1e293b;
  --surface-elevated: #334155;

  --zone-desk: #1e293b;
  --zone-meeting: #1a2744;
  --zone-hotDesk: #1e2433;
  --zone-lounge: #231e33;
  --zone-stroke: #334155;
  --zone-label: #94a3b8;
```

**Step 3: Verify dev server still loads**

Run: `cd /Users/murunkun/MeishuSourceCode/Memoh && pnpm --filter @memoh/web dev`
Expected: Vite starts, no CSS errors

---

### Task 2: Add Deep Space Theme Variables

**Files:**
- Modify: `apps/web/src/style.css`

**Step 1: Add Deep Space theme block**

Add after the `.dark` block, before `@theme inline`:

```css
/* === Deep Space Theme === */
[data-theme="deep-space"] {
  /* Deep space color scale */
  --claw-50:  #e8eaf6;
  --claw-100: #c5cae9;
  --claw-200: #9fa8da;
  --claw-300: #7986cb;
  --claw-400: #5c6bc0;
  --claw-500: #3f51b5;
  --claw-600: #303f9f;
  --claw-700: #252d55;
  --claw-800: #1a2040;
  --claw-900: #0f1629;
  --claw-950: #0a0e1a;

  /* Override shadcn tokens for deep space */
  --background: #0a0e1a;
  --foreground: #f1f5f9;
  --card: #0f1629;
  --card-foreground: #f1f5f9;
  --popover: #0f1629;
  --popover-foreground: #f1f5f9;
  --primary: #06b6d4;
  --primary-foreground: #0a0e1a;
  --secondary: #1a2040;
  --secondary-foreground: #94a3b8;
  --muted: #1a2040;
  --muted-foreground: #64748b;
  --accent: #1a2040;
  --accent-foreground: #94a3b8;
  --destructive: #ef4444;
  --destructive-foreground: #f1f5f9;
  --border: rgba(255, 255, 255, 0.06);
  --input: rgba(255, 255, 255, 0.08);
  --ring: #06b6d4;

  --sidebar: #0f1629;
  --sidebar-foreground: #f1f5f9;
  --sidebar-primary: #06b6d4;
  --sidebar-primary-foreground: #0a0e1a;
  --sidebar-accent: #1a2040;
  --sidebar-accent-foreground: #94a3b8;
  --sidebar-border: rgba(255, 255, 255, 0.06);
  --sidebar-ring: #06b6d4;

  /* OpenClaw design tokens (deep space) */
  --text-primary:   #f1f5f9;
  --text-secondary: #94a3b8;
  --text-muted:     #64748b;
  --text-accent:    #06b6d4;

  --accent-primary:   #06b6d4;
  --accent-secondary: #8b5cf6;
  --accent-success:   #10b981;
  --accent-warning:   #f59e0b;
  --accent-danger:    #ef4444;
  --accent-glow:      rgba(6, 182, 212, 0.2);

  --glass-light:  rgba(255, 255, 255, 0.03);
  --glass-medium: rgba(255, 255, 255, 0.06);
  --glass-heavy:  rgba(255, 255, 255, 0.10);
  --glass-border: rgba(255, 255, 255, 0.06);

  --surface-base:     #0a0e1a;
  --surface-card:     #0f1629;
  --surface-elevated: #1a2040;

  --zone-desk: #1e293b;
  --zone-meeting: #1a2744;
  --zone-hotDesk: #1e2433;
  --zone-lounge: #231e33;
  --zone-stroke: #334155;
  --zone-label: #94a3b8;
}
```

---

### Task 3: Add Glass, Glow, and Animation CSS Classes

**Files:**
- Modify: `apps/web/src/style.css`

**Step 1: Add glass panel classes to `@layer components`**

Add inside the existing `@layer components` block:

```css
  /* Glass panels */
  .glass-panel {
    background: var(--glass-light);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border: 1px solid var(--glass-border);
    border-radius: 0.75rem;
  }
  .glass-panel-hover {
    background: var(--glass-light);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border: 1px solid var(--glass-border);
    border-radius: 0.75rem;
    transition: all 0.2s ease;
  }
  .glass-panel-hover:hover {
    background: var(--glass-medium);
    border-color: var(--accent-primary);
    box-shadow: 0 0 20px var(--accent-glow);
  }
  .glass-panel-solid {
    background: var(--glass-heavy);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1px solid var(--glass-border);
    border-radius: 0.75rem;
  }

  /* Glow borders */
  .glow-border-cyan {
    box-shadow: 0 0 15px rgba(6, 182, 212, 0.15), inset 0 0 15px rgba(6, 182, 212, 0.05);
  }
  .glow-border-purple {
    box-shadow: 0 0 15px rgba(139, 92, 246, 0.15), inset 0 0 15px rgba(139, 92, 246, 0.05);
  }
  .glow-border-green {
    box-shadow: 0 0 15px rgba(16, 185, 129, 0.15), inset 0 0 15px rgba(16, 185, 129, 0.05);
  }
  .glow-border-amber {
    box-shadow: 0 0 15px rgba(245, 158, 11, 0.15), inset 0 0 15px rgba(245, 158, 11, 0.05);
  }

  /* Ambient orbs (Deep Space only) */
  .ambient-orb {
    position: fixed;
    border-radius: 50%;
    filter: blur(80px);
    opacity: 0.12;
    animation: ambient-float 20s ease-in-out infinite;
    pointer-events: none;
    z-index: 0;
  }
  .ambient-orb-cyan {
    width: 600px; height: 600px;
    background: radial-gradient(circle, #06b6d4, transparent);
    top: -200px; right: -100px;
  }
  .ambient-orb-purple {
    width: 500px; height: 500px;
    background: radial-gradient(circle, #8b5cf6, transparent);
    bottom: -150px; left: -100px;
    animation-delay: -10s;
  }

  /* Status dot with glow */
  .status-dot-glow {
    width: 8px; height: 8px;
    border-radius: 50%;
    box-shadow: 0 0 8px currentColor;
    animation: pulse 2s ease-in-out infinite;
  }

  /* Thin scrollbar */
  .scrollbar-thin {
    scrollbar-width: thin;
    scrollbar-color: var(--glass-heavy, rgba(0,0,0,0.12)) transparent;
  }
  .scrollbar-thin::-webkit-scrollbar {
    width: 6px;
  }
  .scrollbar-thin::-webkit-scrollbar-track {
    background: transparent;
  }
  .scrollbar-thin::-webkit-scrollbar-thumb {
    background: var(--glass-heavy, rgba(0,0,0,0.12));
    border-radius: 3px;
  }
  .scrollbar-thin::-webkit-scrollbar-thumb:hover {
    background: var(--text-muted, #94a3b8);
  }

  /* Chat slide animations */
  .animate-slide-up {
    animation: chat-slide-up 200ms ease-out forwards;
  }
  .animate-slide-down {
    animation: chat-slide-down 150ms ease-in forwards;
  }

  /* Deep Space layout overrides */
  [data-theme="deep-space"] .ds-surface-base {
    background-color: var(--surface-base);
  }
  [data-theme="deep-space"] .ds-surface-card {
    background-color: var(--surface-card);
  }
  [data-theme="deep-space"] .ds-border {
    border-color: var(--glass-border);
  }
  [data-theme="deep-space"] .ds-nav-active {
    background-color: rgba(6, 182, 212, 0.1);
    color: var(--accent-primary);
  }
```

**Step 2: Add animation keyframes**

Add before `@layer base`:

```css
/* === Animation Keyframes === */
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
@keyframes agent-pulse {
  0%, 100% { opacity: 0.5; transform: scale(1); }
  50% { opacity: 1; transform: scale(1.08); }
}
@keyframes agent-glow {
  0%, 100% { box-shadow: 0 0 4px currentColor; }
  50% { box-shadow: 0 0 14px currentColor; }
}
@keyframes agent-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}
@keyframes agent-spawn {
  from { transform: scale(0); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}
@keyframes agent-despawn {
  from { transform: scale(1); opacity: 1; }
  to { transform: scale(0); opacity: 0; }
}
@keyframes thinking-dots {
  0%, 80%, 100% { opacity: 0.3; }
  40% { opacity: 1; }
}
@keyframes chat-slide-up {
  from { transform: translateY(100%); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}
@keyframes chat-slide-down {
  from { transform: translateY(0); opacity: 1; }
  to { transform: translateY(100%); opacity: 0; }
}
@keyframes ambient-float {
  0%, 100% { transform: translateY(0px) scale(1); }
  50% { transform: translateY(-30px) scale(1.1); }
}
```

---

### Task 4: Register New Tokens in Tailwind @theme

**Files:**
- Modify: `apps/web/src/style.css`

**Step 1: Add new color tokens to `@theme inline` block**

Add after `--color-sidebar-ring`:

```css
  /* OpenClaw design tokens */
  --color-text-primary: var(--text-primary);
  --color-text-secondary: var(--text-secondary);
  --color-text-muted: var(--text-muted);
  --color-text-accent: var(--text-accent);
  --color-accent-primary: var(--accent-primary);
  --color-accent-secondary: var(--accent-secondary);
  --color-accent-success: var(--accent-success);
  --color-accent-warning: var(--accent-warning);
  --color-accent-danger: var(--accent-danger);
  --color-surface-base: var(--surface-base);
  --color-surface-card: var(--surface-card);
  --color-surface-elevated: var(--surface-elevated);
  --color-glass-light: var(--glass-light);
  --color-glass-medium: var(--glass-medium);
  --color-glass-heavy: var(--glass-heavy);
  --color-glass-border: var(--glass-border);
```

---

### Task 5: Extend Theme Store for Deep Space

**Files:**
- Modify: `apps/web/src/store/settings.ts`

**Step 1: Extend theme type and setTheme logic**

Change theme type from `'light' | 'dark'` to `'light' | 'dark' | 'deep-space'`.

When setting deep-space:
- Apply `.dark` class (since deep-space is a dark variant)
- Apply `data-theme="deep-space"` attribute on `<html>`

When setting light/dark:
- Remove `data-theme` attribute
- Use standard colorMode behavior

---

### Task 6: Update Settings UI with Theme Switcher

**Files:**
- Modify: `apps/web/src/pages/settings/index.vue`
- Modify: `apps/web/src/i18n/locales/en.json`
- Modify: `apps/web/src/i18n/locales/zh.json`

**Step 1: Add Deep Space option to theme selector**

Add a third SelectItem with value "deep-space" and label from i18n.

**Step 2: Add i18n keys**

```json
"themeDeepSpace": "Deep Space"
```
```json
"themeDeepSpace": "深空"
```

---

### Task 7: Add Ambient Orbs to Layout

**Files:**
- Modify: `apps/web/src/pages/main-section/index.vue` or main layout

**Step 1: Add ambient orb elements**

Conditionally render ambient orbs when theme is "deep-space":

```vue
<div v-if="isDeepSpace" class="ambient-orb ambient-orb-cyan" />
<div v-if="isDeepSpace" class="ambient-orb ambient-orb-purple" />
```

---

### Task 8: Verify All Themes Work

**Step 1:** Switch to Light → verify no visual regressions
**Step 2:** Switch to Dark → verify glass variables apply
**Step 3:** Switch to Deep Space → verify ambient orbs, cyan primary, glass effects
**Step 4:** Commit all changes
