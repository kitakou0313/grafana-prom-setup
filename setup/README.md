# refs
- Grafana Tempo tutorial
    - https://grafana.com/docs/tempo/latest/set-up-for-tracing/setup-tempo/deploy/locally/docker-compose/
- Grafana Pyroscope tutorial
    - https://grafana.com/docs/pyroscope/latest/get-started/

# Observability Stack with Docker Compose

This setup includes a complete observability stack with:
- **Grafana**: Visualization and dashboards
- **Prometheus**: Metrics collection and storage
- **Grafana Tempo**: Distributed tracing
- **Grafana Pyroscope**: Continuous profiling

## Directory Structure

```
.
├── docker-compose.yml
├── prometheus/
│   └── prometheus.yml
├── tempo/
│   └── tempo.yml
└── grafana/
    └── provisioning/
        └── datasources/
            └── datasources.yml
```

## Quick Start

1. Start all services:
```bash
docker-compose up -d
```

2. Access the services:
   - Grafana: http://localhost:3000 (default login: admin/admin)
   - Prometheus: http://localhost:9090
   - Tempo: http://localhost:3200
   - Pyroscope: http://localhost:4040

3. Stop all services:
```bash
docker-compose down
```

4. Stop and remove volumes (clean slate):
```bash
docker-compose down -v
```

## Service Details

### Grafana (Port 3000)
- Pre-configured with all datasources
- Anonymous access enabled for testing (disable in production)
- Data sources automatically provisioned

### Prometheus (Port 9090)
- Scrapes metrics from all services every 15 seconds
- Exemplar storage enabled for trace correlation
- Persistent storage in volume

### Tempo (Port 3200, 4317, 4318, 9411)
- OTLP gRPC receiver: 4317
- OTLP HTTP receiver: 4318
- Zipkin receiver: 9411
- Metrics generator enabled with Prometheus integration

### Pyroscope (Port 4040)
- Continuous profiling storage
- Ready to receive profiling data

## Sending Data

### Traces to Tempo (OpenTelemetry)
```bash
# OTLP gRPC endpoint
endpoint: http://localhost:4317

# OTLP HTTP endpoint
endpoint: http://localhost:4318/v1/traces
```

### Metrics to Prometheus
Configure your application to expose metrics and add a scrape config to `prometheus/prometheus.yml`

### Profiles to Pyroscope
```bash
# Example with Go application
import "github.com/grafana/pyroscope-go"

pyroscope.Start(pyroscope.Config{
    ApplicationName: "my-app",
    ServerAddress:   "http://localhost:4040",
})
```

## Customization

- Modify `prometheus/prometheus.yml` to add scrape targets
- Adjust `tempo/tempo.yml` for trace retention and sampling
- Update Grafana provisioning files to add dashboards or modify datasources

## Notes

- All data is persisted in Docker volumes
- Services restart automatically unless stopped
- Network isolation via `observability` bridge network
- Suitable for development/testing (adjust security settings for production)
