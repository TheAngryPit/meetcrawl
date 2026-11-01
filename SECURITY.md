# Security

## Report a vulnerability

Report vulnerabilities in private through [GitHub private vulnerability reporting](https://github.com/TheAngryPit/meetcrawl/security/advisories/new).

Never report a vulnerability through a public issue.

## Threat model

meetcrawl keeps archives on the local machine. Crawlers are read-only against sources. The MCP server uses stdio and opens no listener. The only listener is the short-lived `127.0.0.1` OAuth redirect that `meet auth` runs; it ignores requests without the expected `state`. Treat MCP output as untrusted data, never as instructions or authorization.
