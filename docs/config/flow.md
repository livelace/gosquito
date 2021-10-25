### Flow configuration:

```yaml
flow:
  # Flow parameters.
  name: "flow1"                             # DNS compatible flow name, must be unique.
  params:
    instance: 1                             # How many flow's instances should run in parallel.
                                            # WARNING: Default parallelism is achieved by dedicated flows, 
                                            # not by flow's instance amount. 
                                            # There are no atomic operations over data.
                                            # Use with cautions. 
    
    interval: "5m"                          # How often flow should run (1s minimum).

  # Input plugin parameters:
  # 1. Section is strictly required.
  # 2. Only single plugin is allowed.
  input:
    plugin: "plugin"                        # Input plugin name.
    params:                                 # Input plugin parameters.
    
      # Credentials are very similar to templates, but dedicated for secrets (not showing in logs).
      cred: "creds.input.example"            
    
      # Templates are sections in main configuration file.
      # Templates contain plugin parameters and might be used across different flows.
      # Flow parameters have higher priority over templates.
      # Flow and templates parameters can be used together.
      template: "templates.input.example"    
                                             
                                              
      expire_action: ["/bin/script", "arg"] # Execute command if any plugin source is expired (no new data).
      expire_action_delay: "1d"             # Delay between command executions. 
      expire_action_timeout: 30             # Command execution timeout. 
      expire_interval: "7d"                 # When flow is considered as an expired.
      
      force: true                           # Force fetch data despite new data availability.
      force_count: 10                       # How many data must be fetched from every source.
      
      input: [                              # Plugin sources.
        "izvestia_ru",                      
        "rianru", 
        "tass_agency"
      ]
      
      # Other plugin parameters see on plugin page.
      
  # Process plugins parameters:
  # 1. Section is not strictly required.
  # 2. Multiple plugins allowed.
  process:
    - id: 0                                 # Plugins must be ordered.
      alias: "first step"                   # Stage description.
      plugin: "plugin"                      # Plugin name.
      params:                               
        include: false                      # All filtered/matched/transformed (by this plugin) data will not be 
        ...                                 # included for sending (with output plugin, if declared) by default. 
                                            # Plugin data may be used only by other plugins (require option) and 
                                            # sending is not needed.

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
        include: true                       # Plugin's data will be sent with output plugin.
        require: [1, 0]                     # Require option allows choosing data for processing by this plugin.
        ...                                 # In this example we work with data of two previous plugins. 
                                            # This allows to organize separate processing within one flow.
                                            # WARNING: by default every process plugin works with input plugin data.

  # Output plugin parameters:
  # 1. Section is not strictly required.
  # 2. Only single plugin is allowed.
  output:
    plugin: "plugin"
    params:
      cred: "creds.output.example"
      template: "templates.output.example"      
```

