### Available flow options:

```yaml
flow:
  name: "flow1"                             # DNS compatible flow name.
  params:                                   # Flow parameters.
    interval: "5m"                          # How often flow runs (1 second minimum).

  input:
    plugin: "plugin"                        # Input plugin name.
    params:                                 # Input plugin parameters.
    
      # Credentials are very similar to templates, but dedicated for secret parameters.
      cred: "creds.input.example"            
    
      # Parameters might be set inside flow and/or inside config file template.
      # Flow parameters have higher priority over config template parameters.
      template: "templates.input.example"    
                                              
      expire_action: ["/bin/script", "arg"] # Execute command if any plugin source is expired.
      expire_action_delay: "1d"             # Delay between command executions. 
      expire_action_timeout: 30             # Command execution timeout. 
      expire_interval: "7d"                 # When flow is considered expired.
      
      force: true                           # Force fetch data despite new data availability.
      force_count: 10                       # How many data must be fetched from every plugin source.
      
      input: [                              # Every input plugin can work with multiple sources.
        "izvestia_ru",                      # Every source has its own update/expiration timestamps.
        "rianru", 
        "tass_agency"
      ]
      ...
      
  # Process steps might be not set.    
  process:
    - id: 0                                 # Process plugins must be ordered.
      alias: "first step"                   # Info about current step.
      plugin: "plugin"                      # Process plugin name.
      
      # Process plugin parameters, might or might not contain config template.
      params:                                
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

  # Output step might be not set.
  output:
    plugin: "plugin"
    params:
      cred: "creds.output.example"
      template: "templates.output.example"
      ...
```

