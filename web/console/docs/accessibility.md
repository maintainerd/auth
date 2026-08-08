# Accessibility

## Target

The Maintainerd Auth Console targets **WCAG 2.1 Level AA**. It is the internal
admin surface, so the highest-priority conformance path is the entry point
(login landing) and the shared UI primitives reused across every management
screen.

## Measures in place

- Form fields wire `aria-invalid` + `aria-describedby` to their error/description
  ids across the shared form components (`FormInputField`, `FormPasswordField`,
  `FormSelectField`, `FormCheckboxField`, `FormSwitchField`, `FormDateField`,
  `FormFileUploadField`).
- Errors are surfaced in `role="alert"` regions; icon-only controls carry
  `aria-label`s; the password show/hide toggle is keyboard-reachable.
- Layouts use responsive breakpoints and data tables scroll rather than break.

## Automated verification

Automated accessibility testing (axe scanning) is **deferred to the upcoming
Playwright browser test suite**, to be added by a specialist. Until then,
accessibility is maintained through the measures above plus manual review, and
can be spot-checked with any browser axe extension (start with `/login`).

## Known exceptions / gaps

- No automated a11y gate runs in CI yet — planned as part of the Playwright suite
  (frontend v0.1.0 tracker items G5/K4).
- **The authenticated app** lives under the `/:tenantId` `PrivateLayout` (requires
  a session + tenant), so it needs a live-stack or manual run to audit; it reuses
  the same shared UI primitives as the login landing.
- Third-party WebAuthn / passkey browser prompts are outside the app's control.

Report accessibility issues via the process in [`../SECURITY.md`](../SECURITY.md)
or the repository issue tracker.
