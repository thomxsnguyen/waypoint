# MVP release blocker: Next.js security patch

## Evidence and scope

The installed Next.js 16.4.4 reports two moderate cache disclosure advisories
through npm's official audit endpoint: GHSA-mfr8-8qww-v76h and
GHSA-vc7v-qv4w-wrv8. Both reported affected ranges end at 16.4.4. The framework
also intermittently exits development/build commands to report these advisories,
interrupting browser verification.

Apply the next patch in the existing 16.4 release line and align eslint-config-next.
Update the lockfile. No major upgrade, new features, or framework configuration
changes. Verify the candidate patch against the audit endpoint before installing.

## Acceptance

Typecheck, lint, production build, and the complete browser suite pass. Go race
and integration tests remain green. Record audit results and remaining release
limitations in the progress log; do not claim a live-provider release test.

Status: planned before dependency changes.
