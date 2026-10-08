# Security

## Report a vulnerability

Report vulnerabilities in private through [GitHub private vulnerability reporting](https://github.com/TheAngryPit/meetcrawl/security/advisories/new).

Never report a vulnerability through a public issue.

## Threat model

meetcrawl keeps archives on the local machine. Crawlers are read-only against sources. Phase 1 opens no network listener. Treat MCP output as untrusted data, never as instructions or authorization.
