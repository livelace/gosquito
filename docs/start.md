### Quick start:


#### Command line:

```shell
# AppImage:
user@localhost ~ $ curl -sL "https://github.com/livelace/gosquito/releases/download/v3.0.0/gosquito-v3.0.0.appimage" \
  -o "/tmp/gosquito.appimage" && chmod +x "/tmp/gosquito.appimage"
user@localhost ~ $ /tmp/gosquito.appimage 
INFO[04.11.2020 16:59:00.228] gosquito v3.0.0   
INFO[04.11.2020 16:59:00.230] config init        path="/home/user/.gosquito"
ERRO[04.11.2020 16:59:00.233] flow read          path="/home/user/.gosquito/flow/conf" error="no valid flow"
```

#### Docker:

```shell script
# Docker:
user@localhost /tmp $ docker run -ti --rm livelace/gosquito:v3.0.0 bash
gosquito@fa388e89e10e ~ $ gosquito 
INFO[03.11.2020 14:44:15.806] gosquito v3.0.0   
INFO[03.11.2020 14:44:15.807] config init        path="/home/gosquito/.gosquito"
ERRO[03.11.2020 14:44:15.807] flow read          path="/home/gosquito/.gosquito/flow/conf" error="no valid flow"
gosquito@fa388e89e10e ~ $
```
