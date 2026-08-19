# Security Policy

## Supported Versions

bitown is currently in early development (Phase 1 prototype). Only the latest
commit on `main` is supported.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately via one of the following:

- **GitHub private vulnerability reporting**:  
  [https://github.com/interestic/bitown/security/advisories/new](https://github.com/interestic/bitown/security/advisories/new)
- **Email**: dokumegane@interestic.com

Include:
- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof of concept
- Any suggested mitigations

We will acknowledge receipt within **72 hours** and aim to release a fix within
**14 days** for critical issues.

## Scope

In-scope for this project:

| Area | Examples |
|------|---------|
| API endpoints | Injection, auth bypass, data exposure |
| visites deduplication | Bypass allowing unlimited daily votes |
| Rate limiting | Denial-of-service via API flooding |
| Dependency vulnerabilities | CVEs in Go modules |

Out of scope:

- Issues requiring physical access to the server
- Social engineering attacks
- Findings from automated scanners without a working PoC

## Known Limitations (Phase 1)

- The visitor deduplication key is based on IP + User-Agent hash. It is not
  tamper-proof and determined adversaries can bypass it. This is a known
  trade-off accepted for the prototype phase.
- No rate limiting beyond the per-city-per-day deduplication is currently
  implemented at the API layer.
