### Description:

**rss** plugin is intended for data gathering from RSS/Atom feeds.

### Data structure:

```go
type RssData struct {
	CATEGORIES  []string
	CONTENT     string
	DESCRIPTION string
	GUID        string
	LINK        string
	TITLE       string
}
```

### Config sample:

```toml

```

### Flow sample:

```yaml
flow:
  name: "tass-rss-smtp"

  input:
    plugin: "rss"
    params:
      input: ["http://tass.ru/rss/v2.xml"]

  output:
    plugin: "smtp"
    params:
      template: "templates.rss.smtp.default"
      headers:
        x-gosquito-tag1: "world"
        x-gosquito-tag2: "common"
```