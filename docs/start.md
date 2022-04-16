### Quick start:

gosquito is delivered as [AppImage](https://appimage.org/) package and [Docker image](https://github.com/livelace/gosquito/pkgs/container/gosquito).

#### Command line:

1. Download application:
```shell
# AppImage:
user@localhost ~ $ curl -sL "https://github.com/livelace/gosquito/releases/download/v3.6.0/gosquito-v3.6.0-2158be.appimage" \
  -o "/tmp/gosquito.appimage" && chmod +x "/tmp/gosquito.appimage"
user@localhost ~ $ /tmp/gosquito.appimage 
INFO[30.10.2021 00:27:22.636] gosquito v3.6.0-2158be 
INFO[30.10.2021 00:27:22.636] config apply       path="/home/user/.gosquito"
ERRO[30.10.2021 00:27:22.639] flow read          path="/home/user/.gosquito/conf" error="no valid flow"
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
      match_signature: ["title", "link"]

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
INFO[30.10.2021 00:31:16.468] gosquito v3.6.0-2158be 
INFO[30.10.2021 00:31:16.468] config apply       path="/home/user/.gosquito"
INFO[30.10.2021 00:31:16.471] flow valid         hash="hi7oim" flow="test" file="test.yaml"
INFO[30.10.2021 00:31:16.471] --- flow start     hash="hi7oim" flow="test"
...
```

#### Docker:

```shell
# Docker:
user@localhost ~ $ docker run -ti --rm ghcr.io/livelace/gosquito:v3.6.0
user@04a308454349 ~ $ gosquito
INFO[29.10.2021 21:34:35.915] gosquito v3.6.0-2158be 
INFO[29.10.2021 21:34:35.916] config apply       path="/home/user/.gosquito"
ERRO[29.10.2021 21:34:35.918] flow read          path="/home/user/.gosquito/conf" error="no valid flow"
user@04a308454349 ~ $
```
