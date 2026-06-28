# grafana-prom-setup

Prometheus, Grafanaのsetupを検証する

## テスト用telemetryの生成
```
docker run --network setup_observability --rm ghcr.io/krzko/otelgen:latest -p gprc -i --otel-exporter-otlp-endpoint otel-collector:4317  t multi
```

## ToDo
- Grafana, Prometheus, Loki, Tempoを使った監視PFの構築
    - 1コマンドで構成できるようにする
- OTel operatorでの自動計装
    - https://github.com/open-telemetry/opentelemetry-operator
- Profile周りの調査
    - frame pointers
    - symbol
- テスト用アプリケーションの構築
- OTelで計装して監視