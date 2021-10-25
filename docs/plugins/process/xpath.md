### Description:

**xpath** process plugin is intended for finding HTML elements inside data.


### Generic parameters:

| Param     | Required   | Type    | Templates   | Default   | Example   |
| :-------- | :--------: | :-----: | :---------: | :-------: | :-------: |
| include   | -          | bool    | -           | true      | false     |
| require   | -          | array   | -           | []        | [1, 2]    |


### Plugin parameters:

| Param             | Required   | Type     | Template   | Default   | Example           | Description                                                                                           |
| :---------------- | :--------: | :------: | :--------: | :-------: | :---------------: | :---------------------------------------------------------------------------------------------------- |
| **input**         | +          | array    | -          | []        | ["data.array0"]   | List of [DataItem](../../concept.md) fields with data.                                                |
| **output**        | +          | array    | -          | []        | ["data.array0"]   | List of target [DataItem](../../concept.md) fields.                                                   |
| **xpath**         | +          | array    | +          | []        | ["//a/@href"]     | List of [Xpath](https://en.wikipedia.org/wiki/XPath) queries.                                         |
| xpath_html        | -          | bool     | -          | true      | false             | Get nodes with HTML tags (only text by default).                                                      |
| xpath_html_self   | -          | bool     | -          | true      | false             | Include HTML tags of Xpath node.                                                                      |
| xpath_separator   | -          | string   | -          | "\n"      | false             | Add a custom separator between found nodes.                                                           |

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
      alias: "fetch pages"
      plugin: "fetch"
      params:
        input:  ["rss.link"]
        output: ["data.array0"]

    - id: 1
      alias: "xpath description"
      plugin: "xpath"
      params:
        input:  ["data.array0"]
        output: ["data.array1"]
        xpath:  ["templates.xpath.hh.ru"]

    - id: 2
      alias: "xpath tags"
      plugin: "xpath"
      params:
        input:  ["data.array0"]
        output: ["data.array2"]
        xpath:  ["//span[contains(@data-qa, 'bloko-tag__text')]"]
        xpath_html: false

    - id: 3
      alias: "echo description"
      plugin: "echo"
      params:
        input:  ["data.array1"]
        
    - id: 4
      alias: "echo tags"
      plugin: "echo"
      params:
        input:  ["data.array2"]
```

### Config sample:

```toml
[templates.xpath.hh.ru]
xpath = [
  "//div[contains(@data-qa, 'vacancy-description')]"
]
```

