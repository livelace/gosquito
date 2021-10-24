### Flow configuration:

```yaml
flow:
  # Flow parameters.
  name: "flow1"                             # DNS compatible flow name, must be unique.
  params:
    instance: 1                             # How many flow's instances should run in parallel.
                                            # WARNING: Default parallelism is achieved by dedicated flows, 
                                            # not by flow's instance amount. There are no atomic operations over data.
                                            # Use with cautions. 
    
    interval: "5m"                          # How often flow should run (1s minimum).

  # Input plugin parameters:
  # 1. Strictly required.
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
  # 1. Not strictly required.
  # 2. Multiple plugins allowed.
  process:
    - id: 0                                 # Plugins must be ordered.
      alias: "first step"                   # Step note.
      plugin: "plugin"                      # Plugin name.
      
      params:                               # Plugin parameters, might or might not contain config template.
        include: false                      # Include plugin produced results to output plugin. 
        ...                                 # Not included results can be used inside other plugins. 

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
      
        # Plugin requires data results from Plugin 0 and 1. 
        # Different plugins could require any combinations of plugin results.
        require: [1, 0]
        cred: "creds.process.example"
        template: "templates.process.example"
        ...

  # Output plugin parameters:
  # 1. Not strictly required.
  # 2. Only single plugin is allowed.
  output:
    plugin: "plugin"
    params:
      cred: "creds.output.example"
      template: "templates.output.example"      
```

