# 単一トレースのspanが別々のGrafana Tempoに送信された場合の表示検証レポート

## 目的

マイクロサービス構成などで、同一トレースに属するspanが誤って（あるいは意図せず）別々のGrafana Tempoバックエンドに送信されてしまった場合、Grafana UI（Explore画面）上でそのトレースがどのように見えるかを実際に確認する。

## 検証環境の構成

`experiment/tempo-split-trace/` に、他のいかなるリソースにも依存しない独立したdocker-compose環境を構築した。

- **tempo1** / **tempo2**: `grafana/tempo:latest` を2台、完全に独立したストレージ（それぞれ専用のdocker volume）で起動。相互に一切通信・連携しない。
- **grafana**: `grafana/grafana:latest` を1台起動し、`Tempo1` (→tempo1) と `Tempo2` (→tempo2) の2つのTempoデータソースをprovisioning。
- **service-a** / **service-b**: Go言語で書いた2つの独立したプロセス。
  - `service-a` はHTTPクライアント役。ルートspan `service-a.request` を開始し、OpenTelemetryの `traceparent` ヘッダでトレースコンテキストを伝播しながら `service-b` にHTTPリクエストを送信する。自身のspan（`service-a.request` とその子である `HTTP GET` クライアントspan）は **tempo1** へOTLP exportする。
  - `service-b` はHTTPサーバ役。受信したリクエストから伝播されたトレースコンテキストを抽出し、`GET /handle` サーバspanを生成する。このspanは **tempo2** へOTLP exportする。

これにより、「同一trace ID・正しいparent/child関係を持つが、エクスポート先のバックエンドが異なる」という状況を、実際の2サービス間通信の形で再現した。

## 使用したtrace ID・送信方法

- 実行: `service-b` を先に起動 → `service-a` を実行し、`http://localhost:8081/handle` にGETリクエストを送信
- 生成されたtrace ID: `bbad7217e3e90debfa6b820a874edd6c`
- 送信直後、Tempo API (`/api/traces/{traceID}`) を直接叩いて事前確認したところ、想定通り分割されていた:
  - tempo1側 (`localhost:3200`): `service-a` の2 span（`service-a.request` と `HTTP GET`）
  - tempo2側 (`localhost:3201`): `service-b` の1 span（`GET /handle`、`parentSpanId` は `service-a` 側の `HTTP GET` spanのIDと一致）

## Grafana UI上での表示結果

実装コンテキストを持たない独立した検証者（ブラウザ自動操作エージェント）が、Grafana Explore画面 (`http://localhost:3001/explore`) で各データソースを選択し、TraceQLクエリ `{trace:id="bbad7217e3e90debfa6b820a874edd6c"}` で同一trace IDを検索して得た観察結果。

### Tempo1データソースでの表示

- 検索結果一覧: 1件ヒット。Trace ID `bbad7217e3e90debfa6b820a874edd6c`、Service `service-a`、Name `service-a.request`、Duration `3ms`
- トレース詳細（waterfall）画面:
  - ヘッダ: `service-a: service-a.request 3.14ms`
  - Span Filtersのカウンタ表示: **「2 spans」**
  - span tree: `service-a.request`（親, 3.14ms） → `HTTP GET`（子, 2.96ms）の2 spanのみ。いずれも `service-a` のspan
  - エラーバナー・「no data」表示なし
  - `HTTP GET` クライアントspanで木構造が止まっており、その先（本来子であるはずの `service-b` 側のspan）は一切表示されない

### Tempo2データソースでの表示

- 検索結果一覧: 1件ヒット。Trace ID `bbad7217e3e90debfa6b820a874edd6c`だが、**Service欄が `<root span not yet received>`**、Duration `<1ms` という特殊な表示
- トレース詳細（waterfall）画面:
  - ヘッダ: `service-b: GET /handle 33μs`
  - Span Filtersのカウンタ表示: **「1 spans」**
  - span tree: `GET /handle`（`service-b`）の1 spanのみ、親子関係なし（このバックエンド内では最上位のspanとして描画される）
  - エラーバナー・「no data」表示なし

## 考察

1. **各Tempoは完全にサイロ化されたビューしか持たない**
   2つのTempoインスタンスの間には一切の連携機構がなく、それぞれが「自分が受信したspanだけ」で完結したトレースを再構築して表示する。Tempo1はservice-a側の2 spanだけの（本来より短い）トレースとして、Tempo2はservice-b側の1 spanだけのトレースとして、**互いに矛盾なく・エラーも出さずに**表示してしまう。ユーザーが片方のデータソースしか見ていない場合、トレースが分割されて別バックエンドに存在するという事実に気づく手段がない。

2. **「root span not yet received」というUIヒントが唯一の分割の兆候**
   トレース詳細のwaterfall画面自体にはエラーや欠損を示す表示は一切現れないが、Explore検索結果一覧の段階でのみ、Tempo2側に `<root span not yet received>` というラベルが表示された。これはTempoがトレースの「ルートspan」を把握するための別APIを持っており、それを保持していないバックエンドに対して検索結果一覧上でのみ警告的なラベルを出す、というGrafana Tempo datasourceプラグインの挙動によるものと考えられる。裏を返せば、**ルートspanを保持している側（今回のTempo1）にはこの警告は一切表示されない**ため、「一部だけ見えている」ことに気づくヒントはさらに乏しい。

3. **運用上の含意**
   - otel-collectorのルーティング設定ミス、複数チームでTempoを個別運用している環境、移行作業中の二重書き込みなど、同一トレースのspanが複数バックエンドに分散し得るシナリオは現実的にあり得る。
   - このような状態は、各Tempo単体では「正常に動作しているように見える」（エラーもwarningも出ない）ため、**サービス間のレイテンシ問題やエラーの根本原因調査時に、実際には全体像の一部しか見ていないことに気づかないまま調査が進んでしまうリスク**がある。
   - 唯一の手がかりである `<root span not yet received>` ラベルも、ルートspanを持たない側のデータソースを能動的に確認しない限り目に入らないため、見落としやすい。
   - 対策としては、単一のTempoバックエンド（またはTempoのマルチテナント/フェデレーション機能）にトレースを集約する、あるいはotel-collectorのエクスポート設定を一元管理・監査する運用が望ましい。

## 検証方法の独立性について

上記「Grafana UI上での表示結果」は、環境構築や実装の詳細を一切知らない別セッションのエージェントに、Grafana URL・対象trace ID・データソース名のみを渡して独立に観察させたものである。実装側（本レポート作成者）が期待する結果を伝えていないため、観察結果は実際の挙動をそのまま反映している。
