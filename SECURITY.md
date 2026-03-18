# SECURITY

## Supported Versions

Security fixes are applied to the latest code on the default branch.

## Reporting a Vulnerability

Please do not open a public issue for sensitive security problems.

Recommended process:
1. Prepare a minimal description of the issue, affected endpoints, and impact.
2. Include reproduction steps only when they are necessary to understand the risk.
3. Send the report privately to the project maintainer through GitHub security reporting tools or a private maintainer contact channel.

## Response Goals

- Acknowledge receipt as soon as practical.
- Confirm severity and scope after reproduction.
- Coordinate a fix and release note when the issue is validated.

## Sensitive Areas

Please report these categories privately:
- cookie or credential leakage
- tenant isolation bypass
- account routing confusion across tenants
- SSRF, command execution, or container escape risks
- API endpoints that allow unauthorized publishing or account takeover
