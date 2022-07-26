### Description:

**xpath** process plugin is intended for finding HTML/XML elements inside data (ToDo: Work with broken XML).

### Generic parameters:

| Param   | Required | Type  | Templates | Default | Example |
| :------ | :------: | :---: | :-------: | :-----: | :-----: |
| include |    -     | bool  |     -     |  false  |  true   |
| require |    -     | array |     -     |   []    | [1, 2]  |

### Plugin parameters:

| Param           | Required |  Type  | Template | Default |     Example     | Description                                                                     |
| :-------------- | :------: | :----: | :------: | :-----: | :-------------: | :------------------------------------------------------------------------------ |
| find_all        |    -     |  bool  |    -     |  false  |      true       | Patterns must be found in all selected [Datum](../../concept.md) fields.        |
| **input**       |    +     | array  |    -     |   []    | ["data.array0"] | List of [Datum](../../concept.md) fields with data. Might be text or file path. |
| **output**      |    +     | array  |    -     |   []    | ["data.array0"] | List of target [Datum](../../concept.md) fields.                                |
| **xpath**       |    +     | array  |    +     |   []    |  ["//a/@href"]  | List of [Xpath](https://en.wikipedia.org/wiki/XPath) queries.                   |
| xpath_array     |    -     |  bool  |    -     |  false  |      true       | Put nodes into array (output Datum field must be array).                        |
| xpath_html      |    -     |  bool  |    -     |  true   |      false      | Get nodes with HTML tags.                                                       |
| xpath_html_self |    -     |  bool  |    -     |  true   |      false      | Include HTML tags of Xpath node.                                                |
| xpath_mode      |    -     | string |    -     | "html"  |      "xml"      | Xpath parse mode.                                                               |
| xpath_separator |    -     | string |    -     |   ""    |      "\n"       | Add a custom separator between found nodes.                                     |

### Flow sample:

```yaml
flow:
  name: "xpath-example"

  input:
    plugin: "rss"
    params:
      input:
        [
          "https://spb.hh.ru/search/vacancy/rss?area=113&clusters=true&enable_snippets=true&search_period=1&order_by=publication_time&text=.",
        ]
      force: true
      force_count: 3

  process:
    - id: 0
      plugin: "fetch"
      alias: "fetch pages"
      params:
        input: ["rss.link"]
        output: ["data.text0"]

    - id: 1
      plugin: "xpath"
      alias: "extract xpath"
      params:
        input: ["data.text0", "data.text0"]
        output: ["data.text1", "data.text2"]
        xpath:
          [
            "//div[contains(@data-qa, 'vacancy-description')]",
            "//span[contains(@data-qa, 'bloko-tag__text')]",
          ]
        xpath_html: false

    - id: 2
      alias: "echo data"
      plugin: "echo"
      params:
        input: ["data.text1", "data.text2"]
```
