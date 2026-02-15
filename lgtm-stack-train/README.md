# LGTM Stackの学習

## LGTM Stackとは？
以下のGrafana製品群を使ったO11y構成

- (Grafana)Loki
- Grafana
- (Grafana)Tempo
- (Grafana)Mimir

##　勉強メモ
- Contextは何が起こったのかの詳細を述べる
    - [Semantic Convenctions](https://opentelemetry.io/docs/concepts/semantic-conventions/) に書かれていることは前提として頼りに調査するイメージ

### 始める前ToDo
- UIなどのURL
    - Game UI: http://localhost:8080
    - Grafana: http://localhost:3000
    - Prometheus: http://localhost:9090
    - Alloy Debug: http://localhost:12345/debug/livedebugging
- ダッシュボードセットアップ
    - https://github.com/grafana/alloy-scenarios/tree/main/game-of-tracing#setting-up-the-dashboard

以下はEducational Goalに沿ったまとめ

### Distributed System Architecture
- alloy-scenarios/game-of-tracing/docker-compose.yml をみる

### OpenTelemetry Concepts
- Spanによる関連性
    - 同じServiceのSpanなら同じ色になる
    - Attributesで属性を表示
        - 戦闘の結果
        - 移動した兵隊の数など

### Trace context propagation


## 資料
- ゲームで学ぶTrace
    - こっちをメインに進める
    - docs
        - https://grafana.com/blog/learn-opentelemetry-tracing-through-a-grand-strategy-game-introducing-game-of-traces/
    - repository
        - https://github.com/grafana/alloy-scenarios/tree/main/game-of-tracing
- ゲームで学ぶLGTM Stack
    - https://grafana.com/blog/metrics-logs-traces-and-mayhem-introducing-an-observability-adventure-game-powered-by-grafana-alloy-and-otel/
    - 上の続編
- Grafana tutorial
    - https://grafana.com/tutorials/