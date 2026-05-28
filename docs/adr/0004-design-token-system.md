# 4. Design Token System

Date: 2026-05-28
Status: Accepted

## Context

A consistent visual identity matters disproportionately for a
personal-museum aesthetic: the application is one user's curated
display surface, and inconsistency reads as amateurism. Without
enforced design primitives, components inevitably drift — ad-hoc
hex colors, one-off border radii, bespoke spacing values, and
duplicated chip / button CSS scattered across single-file
components.

**Tailwind CSS was considered and rejected.** Tailwind's utility-class
approach produces template noise that conflicts with the readability
of Vue single-file components, and our palette is small and
opinionated enough (dark theme, gold accent, four category colors,
three metal indicators) to encode directly as CSS custom properties
without a framework.

A token-based system in plain CSS gives us the constraint without
the framework cost.

## Decision

We will encode all design primitives as CSS custom properties in
`src/web/src/assets/styles/variables.css`. Tokens cover:

- **Radii:** `--radius-sm` (8px) | `--radius-md` (12px) |
  `--radius-lg` (16px) | `--radius-full` (9999px)
- **Borders:** `--border-subtle`, `--border-accent`
- **Accents:** `--accent-gold`, `--accent-bronze`,
  `--accent-gold-dim`, `--accent-gold-glow`
- **Backgrounds:** `--bg-card`, `--bg-card-hover`, `--bg-input`
- **Text:** `--text-primary`, `--text-secondary`, `--text-muted`,
  `--text-heading`
- **Category indicators:** `--cat-roman`, `--cat-greek`,
  `--cat-byzantine`, `--cat-modern`
- **Material indicators:** `--mat-gold`, `--mat-silver`,
  `--mat-bronze`
- **Effects:** `--shadow-card`, `--shadow-glow`
- **Motion:** `--transition-fast` (0.2s), `--transition-med` (0.3s)

**All component CSS MUST consume tokens.** Hardcoded colors, radii,
or spacing values are prohibited when a token exists.

Global classes for chips (`.chip`, `.chip-sm`), badges (`.badge`),
buttons (`.btn`, `.btn-sm`, `.btn-xs`, `.btn-primary`,
`.btn-secondary`, `.btn-ghost`, `.btn-danger`), and uppercase
labels (`.section-label`, `.info-label`) live in
`src/web/src/assets/styles/main.css`. Components compose these
classes rather than re-deriving them.

Typography uses a fixed scale: Cinzel for headings (h1 2rem / h2
1.5rem / h3 1.2rem / h4 0.9rem), Inter for body (0.9rem) and
secondary text. Dark theme is the default and only theme.

New components are validated against the token list and the global
class catalog before merge.

## Consequences

+ Visual consistency without coupling components to each other.
+ Theming changes are localised to a single file.
+ Code review can mechanically check token usage — hardcoded hex
  values stand out under grep.
+ Designers can extend the palette without touching component code.
+ Constitution Principle V (Design System) is enforced at the
  source-of-truth layer, not by convention alone.
− Adding a genuinely new visual primitive requires extending the
  token file first — process discipline is non-negotiable.
− New contributors must read the Design System section of
  `.github/copilot-instructions.md` before authoring components;
  the constraint is not self-evident from the codebase alone.
− No utility-class shortcut: rare one-off styling still requires
  composing tokens, which is verbose compared to Tailwind.

## Related

- Constitution Principle V (Design System)
- `.github/copilot-instructions.md` — Design System section
- `src/web/src/assets/styles/variables.css` — token definitions
- `src/web/src/assets/styles/main.css` — global classes
