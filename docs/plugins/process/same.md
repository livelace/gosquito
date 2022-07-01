### Description:

**same** process plugin is intended for matching data similarity.

### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
| :------ | :------: | :---: | :------: | :-----: | :-----: |
| include |    -     | bool  |    -     |  false  |  true   |
| require |    -     | array |    -     |   []    | [1, 2]  |

### Plugin parameters:

| Param           | Required |  Type  | Template |    Default    |     Example      | Description                                                                  |
| :-------------- | :------: | :----: | :------: | :-----------: | :--------------: | :--------------------------------------------------------------------------- |
| **input**       |    +     | array  |    -     |      []       | ["twitter.text"] | List of [Datum](../../concept.md) fields with data.                          |
| same_algo       |    -     | string |    +     | "levenshtein" |      "jaro"      | Similarity [algorithm](https://github.com/hbollon/go-edlib).                 |
| same_all        |    -     |  bool  |    -     |     false     |       true       | Similarity must be matched in all selected [Datum](../../concept.md) fields. |
| same_share_max  |    -     |  int   |    +     |      100      |        70        | Maximum similarity over all data (percents).                                 |
| same_share_min  |    -     |  int   |    +     |       1       |        50        | Minimum similarity over all data (percents).                                 |
| same_ratio_max  |    -     |  int   |    +     |      100      |        70        | Maximum similarity ratio per comparison (percents).                          |
| same_ratio_min  |    -     |  int   |    +     |       1       |        50        | Minimum similarity ratio per comparison (percents).                          |
| same_tokens_max |    -     |  int   |    +     |      100      |       1000       | Maximum amount of tokens for comparison.                                     |
| same_tokens_min |    -     |  int   |    +     |      10       |       100        | Minimum amount of tokens for comparison.                                     |
| same_ttl        |    -     | string |    +     |     "1h"      |      "24h"       | TTL (Time To Live) for saved states (tokens joint into a sentence/state).    |

### Flow sample:

```yaml
flow:
  name: "same-example"

  input:
    plugin: "rss"
    params:
      input: [
        "https://iz.ru/xml/rss/all.xml",
        "https://ria.ru/export/rss2/archive/index.xml",
      ]

  process:
    # rss title may be similar not more than 50% to each (same_share_min: 100) saved state.
    # old states will be wiped in 1 hour.
    - id: 0
      plugin: "same"
      alias: "filter repeated news"
      params:
        input: ["rss.title"]
        same_ratio_max: 50
        same_ratio_min: 1
        same_share_max: 100
        same_share_min: 100
        same_ttl: "1h"

    - id: 1
      plugin: "echo"
      alias: "show unique news"
      params:
        require: [0]
        input: ["rss.title"]
```
