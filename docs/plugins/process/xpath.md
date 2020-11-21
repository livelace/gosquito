### Description:

**xpath** process plugin is intended for finding HTML elements inside
data.


### Generic parameters:

| Param   | Required | Type  | Templates | Default | Example |
|:--------|:--------:|:-----:|:---------:|:-------:|:-------:|
| include |    -     | bool  |     -     |  true   |  false  |
| require |    -     | array |     -     |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Template | Default |     Example     | Description                                                                                         |
|:-----------|:--------:|:-----:|:--------:|:-------:|:---------------:|:----------------------------------------------------------------------------------------------------|
| **input**  |    +     | array |    -     |   []    | ["data.array0"] | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data. |
| **output** |    +     | array |    -     |   []    | ["data.array0"] | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.    |
| **xpath**  |    +     | array |    +     |   []    |  ["//a/@href"]  | List of [Xpath](https://en.wikipedia.org/wiki/XPath) queries.                                       |

### Flow sample:

```yaml
flow:
  name: "xpath-example"

  input:
    plugin: "rss"
    params:
      input: [
        "https://spb.hh.ru/search/vacancy/rss?area=1&clusters=true&enable_snippets=true&search_period=1&specialization=1&text=."
      ]
      force: true
      force_count: 10

  process:
    - id: 0
      alias: "search urls"
      plugin: "regexpfind"
      params:
        input:  ["rss.link"]
        output: ["data.array0"]
        regexp: ["regexps.urls"]

    - id: 1
      alias: "fetch pages"
      plugin: "fetch"
      params:
        input:  ["data.array0"]
        output: ["data.array1"]

    - id: 2
      alias: "xpath href"
      plugin: "xpath"
      params:
        input:  ["data.array1"]
        output: ["data.array2"]
        xpath:  ["templates.xpath.hh.ru"]

    - id: 3
      alias: "echo data"
      plugin: "echo"
      params:
        input:  ["data.array2"]
```

### Config sample:

```toml
[templates.xpath.hh.ru]
xpath = [
  "//div[contains(@data-qa, 'vacancy-description')]", 
  "//span[contains(@data-qa, 'bloko-tag__text')]"
]
```

