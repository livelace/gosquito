### Description:

**xpath** process plugin is intended for finding HTML elements inside data.


### Generic parameters:

| Param   | Required | Type  | Templates | Default | Example |
|:--------|:--------:|:-----:|:---------:|:-------:|:-------:|
| include |    -     | bool  |     -     |  true   |  false  |
| require |    -     | array |     -     |   []    | [1, 2]  |


### Plugin parameters:

| Param           | Required |  Type  | Template | Default |     Example     | Description                                                                        |
|:----------------|:--------:|:------:|:--------:|:-------:|:---------------:|:-----------------------------------------------------------------------------------|
| find_all        |    -     |  bool  |    -     |  false  |      true       | Patterns must be found in all selected [DataItem](../../concept.md) fields.        |
| **input**       |    +     | array  |    -     |   []    | ["data.array0"] | List of [DataItem](../../concept.md) fields with data. Might be text or file path. |
| **output**      |    +     | array  |    -     |   []    | ["data.array0"] | List of target [DataItem](../../concept.md) fields.                                |
| **xpath**       |    +     | array  |    +     |   []    |  ["//a/@href"]  | List of [Xpath](https://en.wikipedia.org/wiki/XPath) queries.                      |
| xpath_html      |    -     |  bool  |    -     |  true   |      false      | Get nodes with HTML tags (only text by default).                                   |
| xpath_html_self |    -     |  bool  |    -     |  true   |      false      | Include HTML tags of Xpath node.                                                   |
| xpath_separator |    -     | string |    -     |  "\n"   |      false      | Add a custom separator between found nodes.                                        |

### Flow sample:

```yaml
flow:
  name: "xpath-example"

  input:
    plugin: "rss"
    params:
      input: [
        "https://spb.hh.ru/search/vacancy/rss?area=113&clusters=true&enable_snippets=true&search_period=1&order_by=publication_time&text=."
      ]
      force: true
      force_count: 3

  process:
    - id: 0
      plugin: "fetch"
      alias: "fetch pages"
      params:
        input:  ["rss.link"]
        output: ["data.text0"]

    - id: 1
      plugin: "xpath"
      alias: "extract xpath"
      params:
        input:  ["data.text0", "data.text0"]
        output: ["data.text1", "data.text2"]
        xpath:  ["//div[contains(@data-qa, 'vacancy-description')]", "//span[contains(@data-qa, 'bloko-tag__text')]"]
        xpath_html: false

    - id: 2
      alias: "echo data"
      plugin: "echo"
      params:
        input:  ["data.text1", "data.text2"]
```

