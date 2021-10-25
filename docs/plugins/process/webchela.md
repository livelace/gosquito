### Description:

**webchela** process plugin is intended for interacting with web pages.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   300   |   900   |

### Plugin parameters:

| Param                  | Required |  Type  | Template |   Default   |              Example              | Description                                    |
|:-----------------------|:--------:|:------:|:--------:|:-----------:|:---------------------------------:|:-----------------------------------------------|
| batch_retry            |    -     |  int   |    +     |      0      |                 3                 | Retry failed batches.                          |
| batch_size             |    -     |  int   |    +     |     100     |                 9                 | Split large amount of URLs into sized batches. |
| browser_argument       |    -     | array  |    +     |     []      |       ["disable-infobars"]        | List of browser arguments.                     |
| browser_extension      |    -     | array  |    +     |     []      |   ["bypass-paywalls-1.7.6.xpi"]   | List of browser extensions.                    |
| browser_geometry       |    -     | string |    +     | "1024x768"  |            "1280x720"             | Browser windows geometry.                      |
| browser_instance       |    -     |  int   |    +     |      1      |                 3                 | Maximum amount of browser instance.            |
| browser_instance_tab   |    -     |  int   |    +     |      5      |                 3                 | Maximum amount of tabs per browser instance.   |
| browser_page_size      |    -     | string |    +     |    "10m"    |               "3m"                | Maximum page size.                             |
| browser_page_timeout   |    -     |  int   |    +     |     20      |                30                 | Maximum time in seconds for page loading.      |
| browser_proxy          |    -     | string |    +     |     ""      |       "http://1.2.3.4:3128"       | Proxy settings (http and socks are supported). |
| browser_script_timeout |    -     |  int   |    +     |     20      |                30                 | Maximum time in seconds for script executions. |
| browser_type           |    -     | string |    +     |  "firefox"  |             "chrome"              | Supported browser types: firefox, chrome.      |
| chunk_size             |    -     | string |    +     |    "3m"     |               "1m"                | Split large messages into sized chunks.        |
| client_id              |    -     | string |    +     | <FLOW_NAME> |          "group1-flow1"           | Custom client identification.                  |
| cpu_load               |    -     |  int   |    +     |     25      |                50                 | Maximum CPU load on a server.                  |
| **input**              |    +     | array  |    +     |     []      |  ["twitter.urls", "data.array0"]  | List of [DataItem](../../concept.md) fields with URLs.             |
| mem_free               |    -     | string |    +     |    "1g"     |               "3g"                | Minimum free MEM size on a server.             |
| output                 |    -     | array  |    +     |     []      |  ["data.array1", "data.array2"]   | List of target [DataItem](../../concept.md) fields.                |
| request_timeout        |    -     |  int   |    +     |     10      |                30                 | Server GRPC request timeout.                   |
| script                 |    -     | array  |    +     |     []      | ["scripts.clicker", "return 42;"] | List of config templates/raw javascript code.  |
| **server**             |    +     | array  |    +     |     []      |   ["server1.example.com:8080"]    | List of Webchela servers.                      |
| server_timeout         |    -     |  int   |    +     |      3      |                10                 | Server connection timeout.                     |

### Flow sample:

```yaml
flow:
  name: "webchela-example"

  input:
    plugin: "rss"
    params:
      input: ["https://tass.ru/rss/v2.xml"]
      force: true
      force_count: 1

  process:
    - id: 0
      alias: "grab pages"
      plugin: "webchela"
      params:
        template: "templates.webchela.default"
        input:  ["rss.link"]
        output: ["data.text0"]

    - id: 1
      plugin: "echo"
      params:
        input: ["data.text0"]
```

### Config sample:

```toml
[templates.webchela.default]
batch_size = 3
browser_type = "firefox"
browser_instance = 1
browser_instance_tab = 3
browser_extension = ["bypass-paywalls-1.7.6.xpi", "ublock-origin-1.30.6.xpi"]
cpu_load = 25
server = ["172.17.0.2:50051"]
timeout = 900
```


