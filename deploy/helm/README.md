# cAdvisor

### Installing the Chart
```helm install cadvisor ./charts/cadvisor```

### Configuration
`disable_metrics` - comma-separated list of `metrics` to be disabled. 

`perf` - configuration of perf events. More information [here](https://github.com/google/cadvisor/blob/master/docs/runtime_options.md#perf-events). 

#### Example perf events configuration:
```yaml
perf:
  core:
    events:
      - ["instructions", "instructions_retired"]
      - "ref-cycles"
    custom_events:
      - type: 4
        config: ["0x5300c0"]
        name: "instructions_retired"
  uncore:
    events:
      - "cas_count_write"
      - "uncore_imc_0/UNC_M_CAS_COUNT:RD"
      - "uncore_ubox/UNC_U_EVENT_MSG"
    custom_events:
      - type: 18
        config: ["0x5300"]
        name: "cas_count_write"
```

### Uninstall the Chart
```helm uninstall cadvisor```