# Grafana, Prometheusに関連する組織について

## Grafana Labs
- Grafanaを開発している企業
- Grafana labsが管理、貢献しているOSS
    - https://grafana.com/oss/?plcmt=oss-nav
    - 管理
        - Grafana, Grafana tempo, Grafana Loki, Grafana Pyroscopeなど
    - 貢献
        - Prometheus
            - CNCF参加

# ライセンスについて
- Prometheus
    - Apache License 2.0
- Grafana, Tempo, Pyroscopeなど
    - AGPLv3

# テスト環境について

## どんな構成にするか
- トレース、メトリクス、プロファイル
- https://opentelemetry.io/docs/demo/

# 資料
- https://github.com/grafana

## デモのアプリケーション
- https://github.com/open-telemetry/opentelemetry-demo

## Grafana Tempo関連
- 設定ファイルのspecification
    - https://grafana.com/docs/tempo/latest/configuration/manifest/
    - https://grafana.com/docs/tempo/latest/configuration/#local-storage-recommendations