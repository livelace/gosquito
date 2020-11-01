### Available flow options:

```yaml
flow:
  name: "flow1"
  params:
    interval: "5m"

  input:
    plugin: "plugin"
    params:
      cred: "creds.input.plugin.example"
      template: "templates.input.plugin.example"
      expire_action: ["/path/to/executable", "arg4", "arg5", "arg6"]
      expire_action_delay: "1d"
      expire_action_timeout: 30
      expire_interval: "7d"
      force: true
      force_count: 10
      ...
      
  process:
    - id: 0
      alias: "first step"
      plugin: "plugin"
      params:
        include: false
        ...

    - id: 1
      alias: "second step"
      plugin: "plugin"
      params:
        include: false
        ...

    - id: 2
      alias: "third step"
      plugin: "plugin"
      params:
        require: [1, 0]
        cred: "creds.process.plugin.example"
        template: "templates.process.plugin.example"
        ...

  output:
    plugin: "plugin"
    params:
      cred: "creds.output.plugin.example"
      template: "templates.output.plugin.example"
      ...
```

