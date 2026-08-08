# Accessibility

## Target

The Maintainerd Auth Identity app targets **WCAG 2.1 Level AA**. Because this is
the public, end-user-facing surface (login, registration, MFA, recovery), the
authentication screens are treated as the highest-priority path for conformance.

## Measures in place

- **Announced errors** — form and auth errors are rendered in `role="alert"` /
  `aria-live` regions (e.g. `LoginForm`, `RegisterForm`, `RegisterInviteForm`,
  `VerifyEmailPage`, `MagicLinkPage`, `OAuthAuthorizePage`), so assistive tech
  announces failures without a focus change.
- **Selected-state semantics** — the MFA method picker (`LoginMFAStep`) exposes
  `aria-pressed` on its option buttons.
- **Labels & keyboard reachability** — inputs are associated with visible labels;
  interactive controls are reachable and operable by keyboard.
- **Responsive** — a `viewport` meta tag is set and layouts use responsive
  breakpoints down to mobile widths.
- **No color-only signaling** — status is conveyed with text/icons in addition to
  color.

## Automated verification

Automated accessibility testing (axe scanning of the auth screens) is **deferred
to the upcoming Playwright end-to-end suite**, which will be added by a Playwright
specialist. Until then, accessibility is maintained through the in-code measures
above plus manual review, and can be spot-checked with any browser axe extension
(scan at minimum `/login`, `/register`, `/forgot-password`).

## Known exceptions / gaps

- No automated a11y gate runs in CI yet — it is planned as part of the Playwright
  suite (see the frontend v0.1.0 tracker items G5/K4).
- Third-party WebAuthn browser prompts are outside the app's control.

Report accessibility issues via the process in [`../SECURITY.md`](../SECURITY.md)
or the repository issue tracker.
