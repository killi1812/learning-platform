---
paths:
  - "**/*.scss"
---
# CSS Guidelines

## Naming Convention

### BEM Structure
- **Block:** `.block-name` — standalone component (use kebab-case)
- **Element:** `.block__element` — part of a block (double underscore)
- **Modifier:** `.block--modifier` — variation of a block or element (double dash)

### Valid Examples
```css
.block
.block__element
.block--modifier
.class-name
```

### Invalid Examples
```css
.Classname      /* No PascalCase */
.className      /* No camelCase */
#block__name    /* No IDs for styling */
```

### JavaScript Hooks
Use `.js-` prefix for JavaScript-only selectors:
```css
.js-trigger-action
```

## Selector Specificity

### Avoid Deep Nesting & High Specificity
```css
/* Bad - overly specific */
#parent .element > ul li a {}
.yara li[class^="accordion-"] .rl-panelbar-counts > span {}

/* Good - flat BEM selectors */
.element-name {}
.rl-panelbar-counts__item {}
```

### Element Hierarchy
Elements should be flat, not nested in class names:
```html
<!-- Bad -->
<span class="block__element__child"></span>

<!-- Good -->
<span class="block__other-element"></span>
```

## SCSS Structure

### Flat Selectors (Good)
```scss
.block {
}
.block__element {
}
```

### Nested Selectors (Bad)
```scss
.block {
  .block__element {
  }
}
```

### Modifier with Parent Reference (Acceptable)
```scss
.block__element {
  some-rule: value;
  
  .block--modifier & {
    some-rule: modified-value;
  }
}
```

## Module Imports

### Use Modern `@use` Syntax
```scss
// Good - modern syntax with namespace
@use "../tools/mixins/px-to-rem" as px-to-rem;
@use "settings.rl.variables" as rl;

// Bad - deprecated @import
@import "../tools/mixins/px-to-rem";
```

## Units & Sizing

### Use `rem()` Function for Pixel Values
Convert pixel values to rem using the `px-to-rem.rem()` function:
```scss
@use "../tools/mixins/px-to-rem" as px-to-rem;

.element {
  height: px-to-rem.rem(40px);
  padding: px-to-rem.rem(10px) px-to-rem.rem(16px);
  font-size: px-to-rem.rem(14px);
}
```

## Variables

### Location
Variables are defined in `tcbase/static/style/settings/settings.rl.variables.scss`.

### Key Variable Categories
- **Colors:** `$red`, `$green`, `$gray`, `$blue`, `$brand-primary`, `$brand-success`, etc.
- **Typography:** `$font-size-base`, `$font-weight-normal`, `$font-stack-primary`
- **Spacing:** `$padding-base-vertical`, `$padding-base-horizontal`
- **Layout:** `$layout-min-width`, `$header-height`, `$sidebar-width`
- **Components:** `$btn-height`, `$input-height`, `$toolbar-height`
- **Z-index:** `$z-index-modal-static`, `$z-index-dropdown`, etc.

### Threat/Classification Colors
```scss
$threat-status-malicious: $red;
$threat-status-suspicious: $orange;
$threat-status-goodware: $green;
$threat-status-unknown: #202020;
```

## Mixins

### Location
Mixins are in `tcbase/static/style/tools/mixins/`.

### Common Mixins
```scss
// Link styling
@include link-variant($color, $decoration, $hover-color, $hover-decoration);

// Classification colors (generates --malicious, --suspicious, etc. modifiers)
@include classification-colors("text");      // for text color
@include classification-colors("background"); // for background color
@include classification-colors("header");    // for header backgrounds
```

## Media Queries

### Use Predefined Breakpoint Variables
```scss
// Component-specific cutoffs
$optimize-summary-header-cutoff: 1220px;
$optimize-sidebar-cutoff: 1400px;

// Screen size breakpoints
$screen-sm-min: 768px;
$screen-md-min: 992px;
$screen-lg-min: 1281px;
$screen-md-max: ($screen-lg-min - 1);
```

### Pattern
```scss
.element {
  font-size: 12px;
  
  @media (min-width: $optimize-summary-header-cutoff) {
    font-size: 16px;
  }
}
```

### Print Styles
Include `@media print` rules for printable components:
```scss
.element {
  padding: 10px;
  
  @media print {
    padding: 8pt;
  }
}
```

## Utility Classes

### Location
Utility classes are in `tcbase/static/style/utility/utility.scss`.

### Common Patterns
- **Display:** `.ui-hide`, `.ui-block`, `.ui-inline-block`, `.hidden`
- **Flexbox:** `.flex`, `.flex-center`, `.flex-space-between`, `.flex-column`
- **Text:** `.text-center`, `.text-right`, `.bold`, `.uppercase`, `.overflow-ellipsis`
- **Width:** `.width-50-pc`, `.width-100-pc`, `.fit-content`
- **Spacing:** `.ml-1`, `.ml-2`, `.margin-auto`
- **Colors:** `.text-color-red`, `.text-color-blue`, `.threat-status--malicious`
- **Cursor:** `.cursor-pointer`, `.cursor-not-allowed`

### Using `@extend` for Utility Inheritance
```scss
.disabled-link-element {
  @extend .text-color-disabled;
  pointer-events: none;
}
```
## Best Practices

1. **Prefer variables over hardcoded values** — use existing variables from `settings.rl.variables.scss`
2. **Use `rem()` for sizing** — ensures consistent scaling
3. **Keep selectors flat** — avoid nesting beyond 2-3 levels
4. **Use BEM modifiers** — not separate classes for variants
5. **Include print styles** — for user-facing components
6. **Avoid `!important`** — exception: utility classes like `.ui-hide`
7. **Use mixins for repeated patterns** — especially `link-variant()` and `classification-colors()`
