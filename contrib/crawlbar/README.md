# CrawlBar manifest

`meetcrawl.json` is a portable `crawlkit.control.v1` manifest. Path fields use `~/…` defaults (Linux XDG layout) that crawlkit expands at runtime. On macOS the binary reports `~/Library/Application Support/meetcrawl/…`, so regenerate the file with the command below instead of copying this one.

Install or refresh for your account:

```bash
meetcrawl metadata --json > ~/.crawlbar/apps/meetcrawl.json
```
