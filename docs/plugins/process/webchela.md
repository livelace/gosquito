### Description:

**webchela** process plugin is intended for interacting with web pages.

### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  false  |  true   |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   300   |   600   |

### Plugin parameters:

| Param                      | Required |  Type  | Template |   Default   |                                          Example                                           | Description                                                                                                                          |
|:---------------------------|:--------:|:------:|:--------:|:-----------:|:------------------------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------------------------------------|
| **input**                  |    +     | array  |    +     |     []      |                              ["twitter.urls", "data.array0"]                               | List of [Datum](../../concept.md) fields with URLs.                                                                                  |
| **output**                 |    -     | array  |    +     |     []      |                               ["data.array1", "data.array2"]                               | List of target [Datum](../../concept.md) fields.                                                                                     |
| **server**                 |    +     | array  |    +     |     []      |                                ["server1.example.com:8080"]                                | List of Webchela servers.                                                                                                            |
| batch_retry                |    -     |  int   |    +     |      0      |                                             3                                              | Retry failed batches.                                                                                                                |
| batch_size                 |    -     |  int   |    +     |     10      |                                             30                                             | Split large amount of URLs into sized batches.                                                                                       |
| browser_argument           |    -     | array  |    +     |     []      |                                    ["disable-infobars"]                                    | List of browser arguments.                                                                                                           |
| browser_extension          |    -     | array  |    +     |     []      |                               ["bypass-paywalls-1.7.6.xpi"]                                | List of browser extensions.                                                                                                          |
| browser_geometry           |    -     | string |    +     | "1920x1080" |                                         "dynamic"                                          | Browser windows geometry (dynamic option makes window maximized.                                                                     |
| browser_instance           |    -     |  int   |    +     |      1      |                                             3                                              | Maximum amount of browser instance.                                                                                                  |
| browser_instance_tab       |    -     |  int   |    +     |     10      |                                             3                                              | Maximum amount of tabs per browser instance.                                                                                         |
| browser_proxy              |    -     | string |    +     |     ""      |                                   "http://1.2.3.4:3128"                                    | Proxy settings (http and socks are supported).                                                                                       |
| browser_type               |    -     | string |    +     |  "chrome"   |                                         "firefox"                                          | Supported browser types: firefox, chrome.                                                                                            |
| chunk_size                 |    -     | string |    +     |    "3m"     |                                            "1m"                                            | Split large messages into sized chunks.                                                                                              |
| client_id                  |    -     | string |    +     | <FLOW_NAME> |                                       "group1-flow1"                                       | Custom client identification.                                                                                                        |
| cookie_input               |    -     | array  |    +     |     []      | ['{"name": "foo", "value": "bar"}'], ["data.text0"], ["data.array0"], ["/tmp/cookie.json"] | JSON string or path to JSON file ([selenium cookie format](https://www.selenium.dev/documentation/webdriver/interactions/cookies/)). |
| cookie_input_file          |    -     |  bool  |    +     |    false    |                                            true                                            | Process cookies as files.                                                                                                            |
| cookie_input_file_mode     |    -     | string |    +     |    text     |                                           lines                                            | Read input file as text or line by line into array.                                                                                  |
| cpu_load                   |    -     |  int   |    +     |     30      |                                             50                                             | Maximum CPU load on a server.                                                                                                        |
| debug_pre_close_delay      |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before close unwanted/unexpected tabs.                                                                      |
| debug_pre_cookie_delay     |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before injecting cookies.                                                                                   |
| debug_pre_open_delay       |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before starting to open tabs.                                                                               |
| debug_pre_process_delay    |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before starting to wait tabs loading.                                                                       |
| debug_pre_screenshot_delay |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before taking screenshots.                                                                                  |
| debug_pre_script_delay     |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before executing scripts.                                                                                   |
| debug_pre_wait_delay       |    -     |  int   |    +     |      0      |                                             10                                             | Time in seconds to delay before starting to process tabs.                                                                            |
| mem_free                   |    -     | string |    +     |    "1g"     |                                            "3g"                                            | Minimum free MEM size on a server.                                                                                                   |
| page_size                  |    -     | string |    +     |    "10m"    |                                            "3m"                                            | Maximum page size.                                                                                                                   |
| page_timeout               |    -     |  int   |    +     |     60      |                                             30                                             | Maximum time in seconds for page loading.                                                                                            |
| retry_codes                |    -     | array  |    -     |     []      |                                         [403, 500]                                         | List of HTTP codes for repeated page loading.                                                                                        |
| retry_codes_tries          |    -     |  int   |    +     |      1      |                                             5                                              | Amount of page reloading tries.                                                                                                      |
| request_timeout            |    -     |  int   |    +     |     10      |                                             30                                             | Server GRPC request timeout.                                                                                                         |
| screenshot_input           |    -     | array  |    -     |     []      |    [["class:super", "css:apple", "id:guid"], ["name=abc", "tag:body", "xpath://html"]]     | List of supported HTML selectors.                                                                                                    |
| screenshot_output          |    -     | array  |    -     |     []      |                               ["data.array0", "data.array1"]                               | List of datums with screenshot paths.                                                                                                |
| screenshot_timeout         |    -     |  int   |    +     |     30      |                                             30                                             | Maximum time in seconds for screenshot elements waiting.                                                                             |
| script_input               |    -     | array  |    -     |     []      |   [["scripts.clicker", "return 42;"], ["return document.documentElement.scrollHeight;"]]   | List of javascript code.                                                                                                             |
| script_output              |    -     | array  |    -     |     []      |                               ["data.array2", "data.array3"]                               | List of datums with corresponding javascript code output.                                                                            |
| script_timeout             |    -     |  int   |    +     |     30      |                                             30                                             | Maximum time in seconds for script execution.                                                                                        |
| server_timeout             |    -     |  int   |    +     |     10      |                                             10                                             | Server connection timeout.                                                                                                           |
| tab_open_randomize         |    -     | string |    +     |    "0:0"    |                                           "3:9"                                            | Random value from range (min:max) in seconds for opening tabs.                                                                       |

### Flow sample:

```yaml
flow:
  name: "webchela-example"
  params:
    interval: "60s"
    cleanup: false

  input:
    plugin: "rss"
    params:
      input: ["https://iz.ru/xml/rss/all.xml"]
      force: true
      force_count: 1

  process:
    - id: 0
      plugin: "webchela"
      params:
        input:  ["rss.link"]
        output: ["data.textA"]

        template: "templates.webchela.default"
        screenshot_input: [["tag:body", "xpath://div[contains(@class, 'top-panel-inside__bottom__inside block-container')]"]]
        screenshot_output: ["data.arrayA"]
        script_input: [["return 42;"]]
        script_output: ["data.arrayB"]

    - id: 1
      plugin: "echo"
      params:
        require: [0]
        input: [
          "data.textA",
          "data.arrayA",
          "data.arrayB",
        ]
```

### Config sample:

```toml
[templates.webchela.default]
batch_size = 10
browser_extension = ["accept-all-cookies-1.0.3.0.crx", "bypass-paywalls-clean-3.7.1.0.crx", "ublock-origin-1.58.0.crx"]
browser_geometry = "dynamic"
browser_instance = 1
browser_instance_tab = 10
browser_type = "chrome"
cookie_input = ['{"name": "foo", "value": "bar"}']
cpu_load = 50
request_timeout = 60
screenshot_timeout = 30
server = ["127.0.0.1:50051"]
timeout = 60
```


