# tempo-split-trace 検証環境

単一トレースのspanが別々のGrafana Tempoに送信されたとき、Grafana UI上でどう表示されるかを試すための使い捨て環境。
`experiment/tempo-split-trace/` 以外のリソースには依存しない（このディレクトリを消せば何も残らない）。

検証結果のまとめは [`../tempo-split-trace-report.md`](../tempo-split-trace-report.md) を参照。

## 構成

- `tempo1` / `tempo2`: 独立したGrafana Tempo（相互連携なし）
- `grafana`: `Tempo1` (→tempo1) / `Tempo2` (→tempo2) の2データソースを持つGrafana
- `service-a`: HTTPクライアント役。ルートspanを開始し `service-b` を呼び出す。自身のspanは **tempo1** へ送信
- `service-b`: HTTPサーバ役。`service-a` から伝播されたtrace contextで子spanを生成。自身のspanは **tempo2** へ送信

## 起動

```sh
cd experiment/tempo-split-trace
docker compose up -d
```

起動確認:

```sh
curl http://localhost:3200/ready   # tempo1
curl http://localhost:3201/ready   # tempo2
curl http://localhost:3001/api/health   # grafana
```

## テストトレースの生成方法

1. `service-b` を先に起動（別ターミナル、または `&` でバックグラウンド実行のままにする）

   ```sh
   go run ./service-b
   ```

2. `service-a` を実行してリクエストを送信。実行のたびに新しいtrace IDが生成され、標準出力に `TRACE_ID=...` として表示される

   ```sh
   go run ./service-a
   ```

   出力例:
   ```
   response from service-b: handled by service-b
   TRACE_ID=bbad7217e3e90debfa6b820a874edd6c
   ```

## ブラウザでの確認方法

Grafana: http://localhost:3001 （匿名ログイン、Adminロールで自動的に入れる）

Explore画面を開き、データソースを `Tempo1` または `Tempo2` に切り替えて、TraceQLクエリで検索する:

```
{trace:id="<TRACE_ID>"}
```

直接開けるURL（`<TRACE_ID>` を実際の値に置換）:

- Tempo1側:
  `http://localhost:3001/explore?schemaVersion=1&panes=%7B%22v6b%22:%7B%22datasource%22:%22tempo1%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22datasource%22:%7B%22type%22:%22tempo%22,%22uid%22:%22tempo1%22%7D,%22queryType%22:%22traceql%22,%22query%22:%22%7Btrace:id=%5C%22<TRACE_ID>%5C%22%7D%22,%22limit%22:20%7D%5D,%22range%22:%7B%22from%22:%22now-1h%22,%22to%22:%22now%22%7D%7D%7D&orgId=1`
- Tempo2側: 上記URL中の `tempo1` を `tempo2` に置換したもの（2箇所）

Tempo APIを直接叩いて確認する場合:

```sh
curl -s "http://localhost:3200/api/traces/<TRACE_ID>" | jq   # tempo1
curl -s "http://localhost:3201/api/traces/<TRACE_ID>" | jq   # tempo2
```

## 後片付け

```sh
cd experiment/tempo-split-trace
docker compose down -v
```
