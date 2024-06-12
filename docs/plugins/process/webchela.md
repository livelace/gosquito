### Description:

**webchela** process plugin is intended for interacting with web pages.

### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  false  |  true   |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   300   |   900   |

### Plugin parameters:

| Param                  | Required |  Type  | Template |   Default   |                                        Example                                         | Description                                               |
|:-----------------------|:--------:|:------:|:--------:|:-----------:|:--------------------------------------------------------------------------------------:|:----------------------------------------------------------|
| **input**              |    +     | array  |    +     |     []      |                            ["twitter.urls", "data.array0"]                             | List of [Datum](../../concept.md) fields with URLs.       |
| **output**             |    -     | array  |    +     |     []      |                             ["data.array1", "data.array2"]                             | List of target [Datum](../../concept.md) fields.          |
| **server**             |    +     | array  |    +     |     []      |                              ["server1.example.com:8080"]                              | List of Webchela servers.                                 |
| batch_retry            |    -     |  int   |    +     |      0      |                                           3                                            | Retry failed batches.                                     |
| batch_size             |    -     |  int   |    +     |     100     |                                           9                                            | Split large amount of URLs into sized batches.            |
| browser_argument       |    -     | array  |    +     |     []      |                                  ["disable-infobars"]                                  | List of browser arguments.                                |
| browser_extension      |    -     | array  |    +     |     []      |                             ["bypass-paywalls-1.7.6.xpi"]                              | List of browser extensions.                               |
| browser_geometry       |    -     | string |    +     | "1024x768"  |                                       "1280x720"                                       | Browser windows geometry.                                 |
| browser_instance       |    -     |  int   |    +     |      1      |                                           3                                            | Maximum amount of browser instance.                       |
| browser_instance_tab   |    -     |  int   |    +     |      5      |                                           3                                            | Maximum amount of tabs per browser instance.              |
| browser_page_size      |    -     | string |    +     |    "10m"    |                                          "3m"                                          | Maximum page size.                                        |
| browser_page_timeout   |    -     |  int   |    +     |     20      |                                           30                                           | Maximum time in seconds for page loading.                 |
| browser_proxy          |    -     | string |    +     |     ""      |                                 "http://1.2.3.4:3128"                                  | Proxy settings (http and socks are supported).            |
| browser_script_timeout |    -     |  int   |    +     |     20      |                                           30                                           | Maximum time in seconds for script executions.            |
| browser_type           |    -     | string |    +     |  "firefox"  |                                        "chrome"                                        | Supported browser types: firefox, chrome.                 |
| chunk_size             |    -     | string |    +     |    "3m"     |                                          "1m"                                          | Split large messages into sized chunks.                   |
| client_id              |    -     | string |    +     | <FLOW_NAME> |                                     "group1-flow1"                                     | Custom client identification.                             |
| cpu_load               |    -     |  int   |    +     |     25      |                                           50                                           | Maximum CPU load on a server.                             |
| mem_free               |    -     | string |    +     |    "1g"     |                                          "3g"                                          | Minimum free MEM size on a server.                        |
| request_timeout        |    -     |  int   |    +     |     10      |                                           30                                           | Server GRPC request timeout.                              |
| screenshot_input       |    -     | array  |    -     |     []      |  [["class:super", "css:apple", "id:guid"], ["name=abc", "tag:body", "xpath://html"]]   | List of supported HTML selectors.                         |
| screenshot_output      |    -     | array  |    -     |     []      |                             ["data.array0", "data.array1"]                             | List of datums with screenshot paths.                     |
| script_input           |    -     | array  |    -     |     []      | [["scripts.clicker", "return 42;"], ["return document.documentElement.scrollHeight;"]] | List of javascript code.                                  |
| script_output          |    -     | array  |    -     |     []      |                             ["data.array2", "data.array3"]                             | List of datums with corresponding javascript code output. |
| server_timeout         |    -     |  int   |    +     |      3      |                                           10                                           | Server connection timeout.                                |

### Flow sample:

```yaml
flow:
  name: "webchela-example"

  input:
    plugin: "rss"
    params:
      input: [ "https://iz.ru/xml/rss/all.xml" ]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "webchela"
      alias: "grab pages"
      params:
        template: "templates.webchela.default"
        input: [ "rss.link" ]
        output: [ "data.text0" ]

    - id: 1
      plugin: "xpath"
      alias: "extract tags"
      params:
        input: [ "data.text0" ]
        output: [ "data.text1" ]
        xpath: [ "//div[contains(@class, 'hash_tags')]" ]
        xpath_html: false

    - id: 2
      plugin: "echo"
      alias: "show data"
      params:
        input: [ "rss.link", "data.text1" ]
```

### Config sample:

```toml
[templates.webchela.default]
batch_size = 10
browser_type = "firefox"
browser_instance = 1
browser_instance_tab = 10
browser_extension = ["bypass-paywalls-1.8.0.xpi", "ublock-origin-1.43.0.xpi"]
cpu_load = 50
server = ["172.17.0.3:50051"]
timeout = 900
```


