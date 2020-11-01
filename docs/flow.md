### Available flow options:

```yaml
flow:
  name: "flow1"                              # DNS compatible flow name.
  params:                                    # Flow parameters.
    interval: "5m"                           # How often flow runs.

  input:
    plugin: "plugin"                         # Input plugin name.
    params:                                  # Input plugin parameters.
      cred: "creds.input.example"            # Credentials from config file.
      template: "templates.input.example"    # Parameters might be set in flow and/or inside config file template.
      expire_action: ["/bin/script", "arg"]
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
        cred: "creds.process.example"
        template: "templates.process.example"
        ...

  output:
    plugin: "plugin"
    params:
      cred: "creds.output.example"
      template: "templates.output.example"
      ...
```

