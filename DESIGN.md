---
name: Kulee
description: A stark industrial control system for dispatching and monitoring queued jobs.
colors:
  ink: "#000000"
  surface: "#ffffff"
  hover-wash: "#f5f5f5"
  canvas: "#e5e5e5"
  disabled-surface: "#d4d4d4"
  dead-fill: "#a3a3a3"
  muted-mark: "#737373"
  muted-text: "#525252"
  failed-fill: "#404040"
  disabled-text: "#262626"
typography:
  display:
    fontFamily: "Arial, Helvetica, sans-serif"
    fontSize: "2.25rem"
    fontWeight: 900
    lineHeight: 1
    letterSpacing: "normal"
  headline:
    fontFamily: "Arial, Helvetica, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 900
    lineHeight: 1.25
    letterSpacing: "normal"
  body:
    fontFamily: "Arial, Helvetica, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  action:
    fontFamily: "Arial, Helvetica, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 900
    lineHeight: 1
    letterSpacing: "normal"
  data:
    fontFamily: "Courier New, Courier, monospace"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  label:
    fontFamily: "Courier New, Courier, monospace"
    fontSize: "0.75rem"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "normal"
rounded:
  square: "0px"
spacing:
  hairline-gap: "2px"
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "20px"
  xl: "28px"
components:
  button-primary:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.surface}"
    typography: "{typography.action}"
    rounded: "{rounded.square}"
    padding: "0 24px"
    height: "44px"
  button-primary-hover:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.action}"
    rounded: "{rounded.square}"
    padding: "0 16px"
    height: "44px"
  button-secondary-hover:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.surface}"
  field:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.data}"
    rounded: "{rounded.square}"
    padding: "0 12px"
    height: "44px"
  status-pending:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "4px 8px"
  status-running:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.surface}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "4px 8px"
  status-success:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "4px 8px"
  status-failed:
    backgroundColor: "{colors.failed-fill}"
    textColor: "{colors.surface}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "4px 8px"
  status-dead:
    backgroundColor: "{colors.dead-fill}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "4px 8px"
---

# Design System: Kulee

## Overview

**Creative North Star: "The Industrial Dispatch Board"**

Kulee is an operational tool, not a marketing surface. It should feel like a durable dispatch board built for fast scanning, direct action, and unambiguous system state. Strong black rules establish structure. White and neutral fills establish hierarchy. Typography carries identity without ornament.

The interface is compact but not cramped. Data can be dense, while primary controls retain a 44px height and compact row actions retain a 36px height. The desktop frame is capped at 80rem. Outer padding is 12px on small screens and 24px from 640px upward. Section padding is 20px on small screens and 28px from 640px upward.

Generic SaaS dashboard styling is the main anti-reference. Kulee must not drift toward floating rounded cards, soft shadows, gradients, decorative accent colors, glass effects, or marketing-style hero metrics. Responsive changes are structural: stack controls, wrap filters, preserve labels, and allow wide data tables to scroll horizontally.

**Key Characteristics:**

- Strict grayscale with no decorative color.
- Square geometry and visible construction.
- Heavy sans headings paired with monospace technical data.
- Ruled sections and tables instead of floating card grids.
- Fast, state-driven interaction with no decorative motion.

**The One Frame Rule.** Extend the main bordered frame with ruled sections before introducing another container. A new card must represent a genuinely independent object, not merely group nearby content.

**The Honest Structure Rule.** Every border, fill, and spacing change must communicate grouping, state, or interaction. Decoration without information is prohibited.

**The 150ms Rule.** Motion is limited to direct hover, focus, selection, disclosure, and overlay feedback, with a maximum duration of 150ms. Loading pulses are allowed. Page entrances, stagger, bounce, parallax, and decorative movement are forbidden. Respect `prefers-reduced-motion` by removing nonessential transitions and pulses.

## Colors

The palette is strict grayscale. Contrast, fill, border style, labels, and position carry meaning together. Color must never become a new decorative layer.

### Primary

- **Ink** (`colors.ink`): Primary text, structural borders, active controls, table headers, and the strongest selected state.
- **Surface** (`colors.surface`): Main content surface, inverse text, fields, and inactive controls.

### Neutral

- **Hover Wash** (`colors.hover-wash`): Row hover and the lightest local interaction feedback.
- **Canvas** (`colors.canvas`): Page background, light metric fills, and the success status fill.
- **Disabled Surface** (`colors.disabled-surface`): Disabled light controls and unavailable states.
- **Dead Fill** (`colors.dead-fill`): Dead status fill.
- **Muted Mark** (`colors.muted-mark`): Noninteractive marks and secondary placeholders where normal text contrast is not required.
- **Muted Text** (`colors.muted-text`): Disabled text and readable secondary copy on light surfaces.
- **Failed Fill** (`colors.failed-fill`): Failed status fill with inverse text.
- **Disabled Text** (`colors.disabled-text`): Strong disabled text on medium neutral fills.

### Named Rules

**The Grayscale Lock Rule.** Future screens may use only the colors in this file. Do not add brand accents, semantic hues, gradients, or tinted neutrals. If a state is unclear, improve its label, fill, border style, icon, or placement instead of adding color.

**The Contrast Rule.** Body text and placeholders must reach at least 4.5:1 against their background. Large text and non-text UI boundaries must reach at least 3:1. Status must never be communicated by fill alone.

**The Status Grammar Rule.** Pending uses a white fill and dashed 2px border. Running uses black fill and white text. Success uses the canvas fill and black text. Failed uses the failed fill and white text. Dead uses the dead fill and black text. Every status includes its uppercase text label.

## Typography

**Display Font:** Arial with Helvetica and sans-serif fallbacks  
**Body Font:** Arial with Helvetica and sans-serif fallbacks  
**Data and Label Font:** Courier New with Courier and monospace fallbacks

**Character:** Arial makes actions and hierarchy blunt and immediate. Courier New marks API paths, IDs, payloads, timestamps, counters, states, and machine-originated messages as technical data. No external font dependency is permitted.

### Hierarchy

- **Display** (`typography.display`): Product identity only. Use 1.875rem below 640px and 2.25rem from 640px upward.
- **Headline** (`typography.headline`): Section headings such as Dispatch job, Queue metrics, and Job ledger.
- **Body** (`typography.body`): Explanations and ordinary interface copy. Long prose is capped at 70ch.
- **Action** (`typography.action`): Buttons and strong controls. Labels are concise and may be uppercase.
- **Data** (`typography.data`): Payloads, job types, IDs, errors, timestamps, and numeric values.
- **Label** (`typography.label`): Table headers, API routes, refresh state, compact metadata, and status labels.

### Named Rules

**The Human and Machine Rule.** Use sans-serif for human navigation, hierarchy, and actions. Use monospace for system output, identifiers, values, routes, and compact operational metadata.

**The Weight Before Size Rule.** Create hierarchy with weight and spacing before increasing type size. Product headings remain fixed-size and restrained. Oversized display copy is prohibited.

**The Uppercase Restraint Rule.** Uppercase is reserved for actions, status labels, table headers, and short machine metadata. Do not place a tiny uppercase eyebrow above every section.

## Elevation

Kulee is flat by default and in every permanent state. It uses no shadows. Depth and grouping come from black 1px and 2px rules, neutral fills, source order, sticky table columns, and temporary stacking only where interaction requires it.

### Shadow Vocabulary

- **None:** `box-shadow` is always `none` for panels, controls, status badges, metric blocks, tables, and rows.

### Named Rules

**The Flat Board Rule.** Permanent surfaces never cast shadows. Use a full border, a tonal fill, or spacing to establish separation.

**The Semantic Stack Rule.** Use stacking only for real overlap such as sticky headers, sticky table columns, menus, confirmations, and toasts. Keep a named order: sticky, dropdown, backdrop, modal, toast, tooltip. Arbitrary values such as 999 are forbidden. `backdrop`, `modal`, `toast`, and `tooltip` are only used where their components are defined (see Confirmation Dialog and Transient Feedback); do not invent ad-hoc overlays.

## Components

Components are square, high-contrast, and explicit. Every interactive component must define default, hover, focus-visible, active, disabled, loading, and error behavior when those states apply.

### Buttons

- **Shape:** Square corners (`rounded.square`) with a 2px ink border.
- **Primary:** Ink fill, surface text, 44px height, and 24px horizontal padding.
- **Primary hover:** Invert to a surface fill and ink text. Active may return to ink fill with no scale or bounce effect.
- **Secondary:** Surface fill, ink text, 44px height, and 16px horizontal padding. Hover inverts to ink fill and surface text.
- **Compact row action:** 36px height and 12px horizontal padding. Use only inside dense tables or toolbars.
- **Focus:** Use a visible 2px outline with a 2px offset. On light surroundings use an ink outline (`focus-visible:outline-black`). On ink fills (primary buttons, active filters) use an inverted surface outline (`focus-visible:outline-white`) with the same 2px offset, because an ink outline is invisible on an ink fill. This inverted treatment is named **Focus Inverse** and applies to every dark-fill control.
- **Disabled:** Keep the label readable, remove hover inversion, and use `cursor: not-allowed`.
- **Loading:** Replace the action label with a direct present-participle state such as ENQUEUING or DELETING. Preserve width where practical.

### Fields

- **Shape:** Square corners with a 2px ink border.
- **Size:** 44px height with 12px horizontal padding.
- **Typography:** Use Data typography for JSON, IDs, routes, job types, and machine-oriented values. Use Body typography for natural-language fields.
- **Focus:** Use the same 2px outline and 2px offset as buttons. Do not rely on a subtle border-color change. Fields always sit on a light surface, so the ink outline applies.
- **Error:** Keep the field visible and place a direct error message nearby. Error copy uses monospace and begins with the failed operation when practical.
- **Native controls:** Keep semantic `input`, `select`, `textarea`, and `button` elements. Do not replace familiar controls for visual novelty.

### Status Badges

- **Shape:** Square corners, 2px border, 4px vertical padding, and 8px horizontal padding.
- **Typography:** Label typography with a 900 weight and uppercase text.
- **Mapping:** Follow the Status Grammar Rule exactly. Pending alone uses a dashed border.
- **Accessibility:** Always render the status word. Fill and border treatment are reinforcement, not the only signal.

### Tables

- **Container:** One 2px ink border around the table region. Use collapsed borders and 1px row separators.
- **Header:** Ink fill, surface text, monospace labels, and 12px cell padding.
- **Rows:** Surface fill at rest and Hover Wash on hover. Do not add a border around every cell.
- **Sticky columns:** Preserve the current surface and hover fill so sticky cells do not become transparent over scrolled content.
- **Responsive behavior:** Keep operational columns intact and scroll the table horizontally. Do not squeeze values into unreadable columns or hide critical state.
- **Loading:** Use a small set of full-row skeletons. Content is visible by default and never depends on an entrance animation.
- **Empty state:** State which filter has no results and provide the next relevant action when one exists. Do not use decorative illustration.

### Metrics

- **Structure:** Use one bordered grid with 2px outer rules and 2px internal gaps created by the ink background.
- **Emphasis:** At most one summary metric may use an ink fill with surface text. Supporting metrics use surface fills.
- **Typography:** Values use heavy monospace. Labels use compact uppercase sans-serif.
- **Responsive behavior:** Use two columns on small screens, three from 640px, and six from 1024px when six metrics are present.
- **Loading and error:** Keep the full metric structure visible. Show placeholders during initial load and a compact error message without replacing the last successful data.

### Ruled Sections

- **Structure:** Separate major areas with a full-width 2px top border inside the main frame.
- **Header row:** Pair a left-aligned section heading with optional right-aligned monospace metadata. Wrap naturally on narrow screens.
- **Spacing:** Use 20px section padding on small screens and 28px from 640px upward.
- **Nesting:** Do not place a bordered card inside another bordered card. Use spacing, a divider, or a table region instead.

### Transient Feedback (Toast)

- **Role:** Report the outcome of a completed action (enqueued, retried, deleted) without replacing the surrounding surface. Not for persistent errors - those stay inline near the failing control.
- **Shape:** Square corners, 2px ink border, surface fill, ink text, and monospace body text.
- **Placement:** Fixed to the bottom-right of the viewport, above the Semantic Stack order at `toast`.
- **Lifetime:** Appear on action completion, auto-dismiss after a few seconds. Dismissal is never the only record: confirmations are also confirmed by the action itself (state change in the table).
- **Stacking:** One toast at a time. A new toast replaces the current one rather than queueing.
- **Motion:** Fade and settle within 150ms. Under `prefers-reduced-motion`, show without entrance motion.
- **Accessibility:** Announce via `aria-live="polite"`.

### Confirmation Dialog

- **Role:** Get explicit consent for destructive, irreversible actions (delete a dead job, delete all dead jobs).
- **Default:** Prefer the native `window.confirm()` - it is a familiar product affordance and requires no custom modal or backdrop. State the affected item and that the action cannot be undone.
- **Custom modal:** Only when a native confirm cannot carry the required information. If built, it follows the Semantic Stack at `modal`/`backdrop`, uses square corners, a 2px ink border, an ink-fill primary confirm, and a surface-fill cancel.
- **Accessibility:** Focus the confirm button on open, trap focus, return focus on close, require an explicit Cancel or outside/ESC dismissal.

### Pagination Controls

- **Role:** Move through cursor-paginated table results. Cursor pagination uses `limit`, `cursor`, and `nextCursor` from the API; there is no page-number navigation.
- **Shape:** Reuse the compact row-action button (36px) in a toolbar beneath the table, or a Secondary button in a small controls row.
- **States:** Next is disabled when there is no `nextCursor`, meaning no more results. Back returns to the previous cursor when one is retained.
- **Metadata:** Show the current result range or the number of records loaded as monospace label text next to the controls.
- **Loading:** Replace the page label with `LOADING` in monospace while fetching; keep controls in place.
- **Empty next:** When a filter or page has no further results, keep the table empty state and disable Next.

## Do's and Don'ts

### Do:

- **Do** preserve strict grayscale and use the named tokens in the frontmatter.
- **Do** use 2px ink borders for primary structure and controls, and 1px rules for row separation.
- **Do** keep primary controls at 44px high and compact table actions at 36px high.
- **Do** pair every state treatment with a readable text label and a strong focus-visible treatment.
- **Do** stack controls and permit table scrolling on narrow screens rather than shrinking text or hiding critical data.
- **Do** use monospace for machine data and sans-serif for human actions and hierarchy.
- **Do** keep loading, empty, disabled, and error states inside the same visual structure as successful content.

### Don't:

- **Don't** imitate a generic SaaS dashboard with floating rounded cards, soft shadows, gradients, decorative accent colors, glassmorphism, or marketing-style hero metrics.
- **Don't** use any corner radius other than 0px, including pills, badges, fields, dialogs, and cards.
- **Don't** use `border-left` or `border-right` greater than 1px as a colored or tonal accent stripe.
- **Don't** use gradient text, decorative grid backgrounds, repeating stripe backgrounds, glow effects, or ornamental blur.
- **Don't** create identical icon-heading-copy card grids when a ruled list, table, or section communicates the content more directly.
- **Don't** add decorative page-load animation, staggered reveals, bounce, elastic motion, parallax, or transitions longer than 150ms.
- **Don't** use display fonts in labels, buttons, tables, or data, and don't add external fonts without revising this system first.
- **Don't** use tiny uppercase eyebrows or numbered markers as default section scaffolding. Numbers are allowed only for a real ordered sequence.
- **Don't** invent custom scrollbars, form controls, or modal behavior for visual flavor. Familiar product affordances take priority.
- **Don't** allow text to overflow its container. Wrap long prose and errors, truncate only when the full value remains available through an accessible detail view or title.
