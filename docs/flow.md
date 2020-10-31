### Available flow options:

```yaml
flow:
  name: "<DNS-COMPATIBLE-FLOW-NAME>"
  params:
    interval: "5m"

  input:
    plugin: "<INPUT_PLUGIN_NAME>"
    params:
      cred: "<CREDENTIALS.INPUT.PLUGIN.EXAMPLE>"
      template: "<TEMPLATES.INPUT.PLUGIN.EXAMPLE>"
      expire_action: ["/path/to/executable", "arg4", "arg5", "arg6"]
      expire_action_delay: "1d"
      expire_action_timeout: 30
      expire_interval: "7d"
      input: ["<FEED>", "<CHANNEL>", "<CHAT>"]
      force: true
      count: 10
      

  process:
    - id: 0
      alias: "<CLEAR_DESCRIPTION>"
      plugin: "<PROCESS_PLUGIN_NAME>"
      params:
        include: false
        input:  ["<PLUGIN_PARAMS>"]

    - id: 1
      plugin: "<PROCESS_PLUGIN_NAME>"
      params:
        include: false
        input:  ["<PLUGIN_PARAMS>"]

    - id: 2
      plugin: "<PROCESS_PLUGIN_NAME>"
      params:
        require: [1, 0]
        cred: "<CREDENTIALS.PROCESS.PLUGIN.EXAMPLE>"
        template: "<TEMPLATES.PROCESS.PLUGIN.EXAMPLE>"
        input:  ["<PLUGIN_PARAMS>"]
        output: ["<PLUGIN_PARAMS>"]

  output:
    plugin: "<OUTPUT_PLUGIN_NAME>"
    params:
      cred: "<CREDENTIALS.OUTPUT.PLUGIN.EXAMPLE>"
      template: "<TEMPLATES.OUTPUT.PLUGIN.EXAMPLE>"
```

