### Description:

**regexpfind** process plugin is intended for finding patterns inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example | Description |
|:--------|:--------:|:-----:|:-------:|:-------:|:------------|
| include |    -     | bool  |  true   |  false  |             |
| require |    -     | array |   []    | [1, 2]  |             |


### Plugin parameters:

| Param  | Required | Type  | Default |         Example          | Description |
|:-------|:--------:|:-----:|:-------:|:------------------------:|:------------|
| input  |    +     | array |   []    |     ["twitter.urls"]     |             |
| output |    +     | array |   []    |     ["data.array0"]      |             |
| regexp |    +     | array |   []    | ["http://go.tass.ru/.*"] |             |

### Config sample:

```toml

```

### Flow sample:

```yaml
```