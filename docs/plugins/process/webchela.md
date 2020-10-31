### Description:

**smtp** output plugin is for sending data as emails.


### Generic parameters:

| Param   | Required | Type  | Default | Example | Description |
|:--------|:--------:|:-----:|:-------:|:-------:|:------------|
| include |    -     | bool  |  true   |  false  |             |
| require |    -     | array |   []    | [1, 2]  |             |


### Plugin parameters:

| Param                  | Required |  Type  | Template |   Default   |              Example              | Description |
|:-----------------------|:--------:|:------:|:--------:|:-----------:|:---------------------------------:|:------------|
| batch_retry            |    -     |  int   |    +     |      0      |                 3                 |             |
| batch_size             |    -     |  int   |    +     |     100     |                 9                 |             |
| browser_argument       |    -     | array  |    +     |     []      |       ["disable-infobars"]        |             |
| browser_extension      |    -     | array  |    +     |     []      |   ["bypass-paywalls-1.7.6.xpi"]   |             |
| browser_geometry       |    -     | string |    +     | "1024x768"  |            "1280x720"             |             |
| browser_instance       |    -     |  int   |    +     |      1      |                 3                 |             |
| browser_instance_tab   |    -     |  int   |    +     |      5      |                 3                 |             |
| browser_page_size      |    -     | string |    +     |    "10m"    |               "3m"                |             |
| browser_page_timeout   |    -     |  int   |    +     |     20      |                30                 |             |
| browser_script_timeout |    -     |  int   |    +     |     20      |                30                 |             |
| browser_type           |    -     | string |    +     |  "firefox"  |             "chrome"              |             |
| chunk_size             |    -     | string |    +     |    "3m"     |               "1m"                |             |
| client_id              |    -     | string |    +     | <flow_name> |          "group1-flow1"           |             |
| cpu_load               |    -     |  int   |    +     |     25      |                50                 |             |
| input                  |    +     | array  |    +     |     []      |  ["data.array0", "data.array1"]   |             |
| mem_free               |    -     | string |    +     |    "1g"     |               "3g"                |             |
| output                 |    -     | array  |    +     |     []      |  ["data.array2", "data.array3"]   |             |
| request_timeout        |    -     |  int   |    +     |     10      |                30                 |             |
| script                 |    -     | array  |    +     |     []      | ["scripts.clicker", "return 42;"] |             |
| server                 |    +     | array  |    +     |     []      |   ["server1.example.com:8080"]    |             |
| server_timeout         |    -     |  int   |    +     |      3      |                10                 |             |
| timeout                |    -     |  int   |    +     |     300     |                900                |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```