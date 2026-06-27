# refs
- Grafana Tempo tutorial
    - https://grafana.com/docs/tempo/latest/set-up-for-tracing/setup-tempo/deploy/locally/docker-compose/
- Grafana Pyroscope tutorial
    - https://grafana.com/docs/pyroscope/latest/get-started/
- OpenTelemetry Collector documentation
    - https://opentelemetry.io/docs/collector/

# Observability Stack with Docker Compose

This setup includes a complete observability stack with:
- **OpenTelemetry Collector**: Signal gateway — receives traces and metrics from applications and forwards them to the appropriate backends
- **Grafana**: Visualization and dashboards
- **Prometheus**: Metrics collection and storage
- **Grafana Tempo**: Distributed tracing
- **Grafana Pyroscope**: Continuous profiling

## Architecture

```
Application
  │
  ├─ Traces  (OTLP gRPC :4317 / HTTP :4318)
  ├─ Metrics (OTLP gRPC :4317 / HTTP :4318)
  │
  ▼
OTel Collector
  │
  ├─ Traces  ──► Tempo        (internal :4317)
  └─ Metrics ──► Prometheus   (remote write :9090)

Application
  └─ Profiles ──► Pyroscope   (:4040, direct)
```

> Profiles are sent directly to Pyroscope because the OpenTelemetry profiling signal
> is not yet stable and standard OTel Collector support is limited.

## Directory Structure

```
.
├── docker-compose.yaml
├── otel-collector/
│   └── otel-collector.yaml
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

### OpenTelemetry Collector (Ports 4317, 4318, 8888)
- OTLP gRPC receiver: 4317 (from apps)
- OTLP HTTP receiver: 4318 (from apps)
- Collector self-metrics: 8888 (scraped by Prometheus)
- Forwards traces to Tempo via OTLP gRPC
- Forwards metrics to Prometheus via remote write

### Grafana (Port 3000)
- Pre-configured with all datasources
- Anonymous access enabled for testing (disable in production)
- Data sources automatically provisioned

### Prometheus (Port 9090)
- Scrapes metrics from Prometheus itself and OTel Collector every 15 seconds
- Receives metrics forwarded from OTel Collector via remote write
- Exemplar storage enabled for trace correlation
- Persistent storage in volume

### Tempo (Port 3200)
- Query HTTP API: 3200
- OTLP ports are internal only (OTel Collector forwards to Tempo on the internal network)
- Metrics generator enabled with Prometheus integration

### Pyroscope (Port 4040)
- Continuous profiling storage
- Receives profiling data directly from applications

## Sending Data

All signals except profiling go through the OTel Collector.

### Traces and Metrics (via OTel Collector)

Configure your application's OTel SDK to export to the OTel Collector:

```yaml
# OTLP gRPC endpoint
endpoint: http://localhost:4317

# OTLP HTTP endpoint
endpoint: http://localhost:4318
```

Example (environment variables):
```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

### Profiles to Pyroscope (direct)
```go
// Example with Go application
import "github.com/grafana/pyroscope-go"

pyroscope.Start(pyroscope.Config{
    ApplicationName: "my-app",
    ServerAddress:   "http://localhost:4040",
})
```

## Customization

- Modify `otel-collector/otel-collector.yaml` to add processors (sampling, filtering) or additional exporters
- Modify `prometheus/prometheus.yml` to add extra scrape targets
- Adjust `tempo/tempo.yml` for trace retention and sampling
- Update Grafana provisioning files to add dashboards or modify datasources

## Notes

- All data is persisted in Docker volumes
- Services restart automatically unless stopped
- Network isolation via `observability` bridge network
- Suitable for development/testing (adjust security settings for production)
