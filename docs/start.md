### Quick start:

gosquito is delivered as [AppImage](https://appimage.org/) package and [Docker image](https://github.com/livelace/gosquito/pkgs/container/gosquito).

#### Command line:

1. Download application:
```shell
# AppImage:
user@localhost ~ $ curl -sL "https://github.com/livelace/gosquito/releases/download/v4.5.0/gosquito-v4.5.0-385342.appimage" \
  -o "/tmp/gosquito.appimage" && chmod +x "/tmp/gosquito.appimage"
user@localhost ~ $ /tmp/gosquito.appimage 
INFO[18.06.2024 00:27:22.636] gosquito v4.5.0-385342
INFO[18.06.2024 00:27:22.636] config apply       path="/home/user/.gosquito"
ERRO[18.06.2024 00:27:22.639] flow read          path="/home/user/.gosquito/conf" error="no valid flow"
```

2. Save flow example (~/.gosquito/conf/test.yaml):
```yaml
flow:
  name: "test"

  input:
    plugin: "rss"
    params:
      force: true
      force_count: 10
      input: ["https://www.opennet.ru/opennews/opennews_all.rss"]
      match_signature: ["rss.link", "rss.title"]

  process:
    - id: 0
      alias: "echo title"
      plugin: "echo"
      params:
        input: ["rss.title"]
```

3. Run application:

```shell
user@localhost ~ $ /tmp/gosquito.appimage 
INFO[18.06.2024 00:27:22.636] gosquito v4.5.0-385342
INFO[18.06.2024 00:27:22.636] config apply       path="/home/user/.gosquito"
ERRO[18.06.2024 00:27:22.639] flow read          path="/home/user/.gosquito/conf" error="no valid flow"
...
```

#### Docker:

```shell
# Docker:
user@localhost ~ $ docker run -ti --rm ghcr.io/livelace/gosquito:v4.5.0
user@04a308454349 ~ $ gosquito
INFO[18.06.2024 00:27:22.636] gosquito v4.5.0-385342
INFO[18.06.2024 00:27:22.636] config apply       path="/home/user/.gosquito"
ERRO[18.06.2024 00:27:22.639] flow read          path="/home/user/.gosquito/conf" error="no valid flow"
user@04a308454349 ~ $
```
