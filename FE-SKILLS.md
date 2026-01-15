Note: this should go in .claude/skills/frontend-design/SKILL.md
# Frontend Design & Building Skill

Create distinctive, production-grade frontend interfaces with high design quality. Use this skill when building web components, pages, applications, or any user-facing interface.

## When to Use This Skill

- Building landing pages, dashboards, or web applications
- Creating React/Vue/Svelte components
- Designing forms, modals, cards, or any UI elements
- Styling existing interfaces
- Building artifacts in Claude.ai

## Core Principles

### 1. Avoid Generic AI Aesthetics

Never default to these overused patterns:
- **Fonts**: Roboto, Arial, system fonts
- **Colors**: Purple gradients on white backgrounds, generic blue CTAs
- **Layouts**: Predictable hero-features-testimonials-footer patterns
- **Effects**: Subtle shadows everywhere, 4px border-radius on everything

### 2. Commit to a Bold Aesthetic Direction

Before writing any code, choose a clear design direction:

| Direction | Characteristics |
|-----------|-----------------|
| Brutalist/Raw | Exposed structure, monospace fonts, harsh contrasts, intentional roughness |
| Luxury/Refined | Generous whitespace, serif typography, muted palettes, subtle animations |
| Retro-Futuristic | Neon accents, dark backgrounds, geometric shapes, tech-inspired fonts |
| Editorial/Magazine | Strong typography hierarchy, asymmetric layouts, dramatic imagery |
| Organic/Natural | Soft curves, earth tones, flowing animations, handwritten elements |
| Playful/Toy-like | Bold colors, rounded shapes, bouncy animations, oversized elements |
| Industrial/Utilitarian | Monochrome, grid-heavy, functional typography, minimal decoration |
| Art Deco/Geometric | Gold accents, symmetry, decorative patterns, elegant serif fonts |

### 3. Typography That Stands Out

Choose fonts with character. Pair a distinctive display font with a refined body font:

**Display fonts to consider**: Space Grotesk, Clash Display, Cabinet Grotesk, Satoshi, General Sans, Plus Jakarta Sans, Syne, Outfit, Manrope, DM Sans

**Serif alternatives**: Fraunces, Playfair Display, Crimson Pro, Source Serif Pro, Lora

**Monospace for tech**: JetBrains Mono, Fira Code, IBM Plex Mono, Source Code Pro

Load from Google Fonts or use `@font-face` with self-hosted files.

### 4. Color With Intention

- **Dark themes**: Not just `#1a1a1a` — try deep blues, warm blacks, or tinted darks
- **Light themes**: Not just white — consider warm off-whites, cool grays, or tinted backgrounds
- **Accent colors**: One bold accent, used sparingly but consistently
- **Gradients**: If using gradients, make them purposeful and unique (mesh gradients, radial fades, directional sweeps)

### 5. Atmospheric Backgrounds

Replace solid colors with:
- Layered CSS gradients
- Subtle geometric patterns
- Grain/noise textures (CSS or SVG)
- Glassmorphism with meaningful blur
- Contextual imagery or illustrations

### 6. Purposeful Animation

Focus on high-impact moments rather than scattered micro-interactions:

```css
/* Staggered entrance animation */
.item { 
  animation: fadeSlideIn 0.6s ease-out forwards;
  animation-delay: calc(var(--index) * 0.1s);
}

@keyframes fadeSlideIn {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}
```

Use `animation-delay` for orchestrated reveals. Prefer CSS transitions over JavaScript animation libraries for simple effects.

## Implementation Guidelines

### File Structure

For complex frontends, prefer multiple files:

```
src/
├── components/
│   ├── Button.tsx
│   ├── Card.tsx
│   └── Layout.tsx
├── styles/
│   ├── globals.css
│   ├── variables.css
│   └── animations.css
└── pages/
    └── index.tsx
```

For artifacts or single-file outputs, structure clearly with comments:

```html
<!-- Component: Hero Section -->
<!-- Component: Features Grid -->
<!-- Styles -->
<style>...</style>
<!-- Scripts -->
<script>...</script>
```

### CSS Architecture

Use CSS custom properties for theming:

```css
:root {
  /* Spacing scale (8px base) */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;
  --space-12: 3rem;
  --space-16: 4rem;

  /* Typography scale */
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;
  --text-2xl: 1.5rem;
  --text-3xl: 2rem;
  --text-4xl: 2.5rem;

  /* Colors */
  --color-bg: #0a0a0f;
  --color-surface: #16161d;
  --color-border: #2a2a35;
  --color-text: #e4e4e7;
  --color-text-muted: #71717a;
  --color-accent: #6366f1;
  --color-accent-hover: #818cf8;

  /* Effects */
  --radius-sm: 0.375rem;
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.1);
  --shadow-md: 0 4px 12px rgba(0,0,0,0.15);
  --transition-fast: 150ms ease;
  --transition-base: 200ms ease;
}
```

### Accessibility Requirements

- Color contrast ratio: 4.5:1 minimum for text
- Focus states: Visible and distinctive (not just outline)
- Semantic HTML: Use proper heading hierarchy, landmarks, and ARIA when needed
- Keyboard navigation: All interactive elements must be reachable
- Motion: Respect `prefers-reduced-motion`

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

### Performance Considerations

- Lazy load images below the fold
- Use `loading="lazy"` and `decoding="async"` on images
- Prefer CSS for animations over JavaScript
- Minimize layout shifts (set explicit dimensions on media)
- Use `font-display: swap` for web fonts

### React/Component Patterns

```tsx
// Prefer composition over configuration
<Card>
  <Card.Header>
    <Card.Title>Title</Card.Title>
  </Card.Header>
  <Card.Body>Content</Card.Body>
</Card>

// Use CSS modules or styled-components for scoped styles
import styles from './Button.module.css';

// Implement proper TypeScript interfaces
interface ButtonProps {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  children: React.ReactNode;
  onClick?: () => void;
}
```

## Checklist Before Delivering

- [ ] Aesthetic direction is clear and consistent
- [ ] No generic fonts (Inter, Roboto, Arial)
- [ ] Colors are intentional, not default blue/purple
- [ ] Background has depth (not plain white/gray)
- [ ] Typography hierarchy is clear
- [ ] Animations are purposeful, not scattered
- [ ] Responsive design works on mobile
- [ ] Accessibility basics are covered
- [ ] Code is clean and well-organized

## References

- [Anthropic Frontend Design Plugin](https://github.com/anthropics/claude-code/tree/main/plugins/frontend-design)
- [Frontend Aesthetics Cookbook](https://github.com/anthropics/claude-code/blob/main/plugins/frontend-design/skills/frontend-design/SKILL.md)
- [Google Fonts](https://fonts.google.com)
- [Coolors](https://coolors.co) for palette generation
- [Contrast Checker](https://webaim.org/resources/contrastchecker/)