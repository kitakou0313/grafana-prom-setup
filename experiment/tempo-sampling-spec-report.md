# OpenTelemetryのtrace samplingの仕様調査レポート

## 目的

[`tempo-split-trace-report.md`](./tempo-split-trace-report.md) では、「同一トレースのspanが複数のTempoバックエンドに分割してexportされた場合、Grafana UI上でどう見えるか」を検証した。その検証環境では両サービスとも常に100%のspanを記録・exportしており、samplingによってspanが欠落するケースは扱っていない。

本レポートはそれと切り離した別テーマとして、OpenTelemetryのtrace samplingそのものの仕様を調査する。具体的には以下2点を明らかにする。

1. samplingの判断は「トレース単位」で行われるのか、それとも「spanごと」に個別に判断されるのか
2. 「トレース全体がまるごとサンプリングされない」のではなく、「同一トレース内の一部のspanだけがサンプリングされない（記録・exportされない）」ということが起こり得るか

## 参照した資料

調査にあたっては以下の3つの資料のみを参照した。

1. OpenTelemetry公式ドキュメント「Sampling」概念ページ: https://opentelemetry.io/docs/concepts/sampling/
2. このリポジトリで実際に使われているOpenTelemetry SDK
   - `experiment/tempo-split-trace/` の `service-a`・`service-b`（Go実装）が依存する `go.opentelemetry.io/otel/sdk v1.45.0`（[`go.mod`](./tempo-split-trace/go.mod)）
   - ローカルのGoモジュールキャッシュ（`$GOPATH/pkg/mod/go.opentelemetry.io/otel/sdk@v1.45.0/trace/`）にソース一式が存在するため、`tracer.go` / `sampling.go` / `provider.go` / `sampler_env.go` を直接参照した
3. W3C Trace Context仕様: https://www.w3.org/TR/trace-context/

## 結論のサマリ

- SDKの実装としては、samplingの判断（`Sampler.ShouldSample`）は **spanが1つ生成されるたびに毎回呼ばれる**。「トレース開始時に1回だけ判断してそれ以降は伝播するだけ」という保証はAPI設計上はない。
- しかし、SDKの**デフォルトのサンプラー設定**（`ParentBased(AlwaysSample())`）と、W3C Trace Contextの`sampled` flag伝播の組み合わせにより、**素直な構成であれば結果的にトレース全体で一貫した判断になる**。
- ただし、これはあくまで「素直な構成の場合の結果」であって仕組み上の保証ではなく、**カスタムサンプラーやサービスごとに異なるサンプラー設定を使えば、同一トレース内の一部のspanだけが記録・exportされない状況は技術的に起こり得る**。W3Cの`sampled` flagも「推奨であり強制ではない」と明記されている。
- 今回のリポジトリの検証環境（`tempo-split-trace`）は、両サービスともサンプラー未設定＝デフォルトの`ParentBased(AlwaysSample())`で常に100%記録されており、そこで観測された「トレースの分割」はsamplingによる欠落ではなく、記録済みspanのexport先（バックエンド）ルーティングが分かれていたことによるものである。

## 詳細

### 1. サンプリング判断は「spanごと」に行われる（SDK実装の事実）

`go.opentelemetry.io/otel/sdk@v1.45.0/trace/tracer.go` の `tracer.newSpan` では、spanを1つ生成するたびに以下のようにサンプラーが呼ばれる。

```go
samplingResult := tr.provider.sampler.ShouldSample(SamplingParameters{
    ParentContext: ctx,
    TraceID:       tid,
    Name:          name,
    Kind:          config.SpanKind(),
    Attributes:    config.Attributes(),
    Links:         config.Links(),
})
...
if !isRecording(samplingResult) {
    return tr.newNonRecordingSpan(sc)
}
return tr.newRecordingSpan(ctx, psc, sc, name, samplingResult, config)
```

`SamplingParameters` にはトレースIDだけでなく、span名・`SpanKind`・attributes・`Links`・親のcontextが渡される。つまりサンプラーは理屈の上では、同じトレースIDに属するspanであっても、span名やattributesを見て個別に異なる判断（`Drop` / `RecordOnly` / `RecordAndSample`、`trace/sampling.go` の `SamplingDecision`）を返すことができる。この時点では「トレース単位」であることはAPIとして保証されていない。

これはOpenTelemetry公式ドキュメントの記述とも整合する。同ページはHead Samplingについて「a sampling decision as early as possible」と述べており、判断のタイミングそのものについて「トレースにつき1回」という制約は明言していない。

### 2. デフォルト設定では、結果的に「トレース全体」で一貫した判断になる

`go.opentelemetry.io/otel/sdk@v1.45.0/trace/provider.go` では、`WithSampler` が明示的に指定されない場合、以下のようにデフォルトサンプラーがセットされる。

```go
if cfg.sampler == nil {
    cfg.sampler = ParentBased(AlwaysSample())
}
```

（`OTEL_TRACES_SAMPLER` / `OTEL_TRACES_SAMPLER_ARG` 環境変数が設定されていれば `samplerFromEnv()` がそちらを優先する。`trace/sampler_env.go` 参照。）

`ParentBased` サンプラーの実体（`trace/sampling.go`）は次の通り。

```go
func (pb parentBased) ShouldSample(p SamplingParameters) SamplingResult {
    psc := trace.SpanContextFromContext(p.ParentContext)
    if psc.IsValid() {
        if psc.IsRemote() {
            if psc.IsSampled() {
                return pb.config.remoteParentSampled.ShouldSample(p)
            }
            return pb.config.remoteParentNotSampled.ShouldSample(p)
        }
        if psc.IsSampled() {
            return pb.config.localParentSampled.ShouldSample(p)
        }
        return pb.config.localParentNotSampled.ShouldSample(p)
    }
    return pb.root.ShouldSample(p)
}
```

つまり「親spanがない（root span）場合のみ委譲先サンプラー（デフォルトは`AlwaysSample()`）で判断し、親がある場合は親のsampled flagをそのまま継承する」実装になっている。デフォルト設定（`remoteParentSampled` / `localParentSampled` いずれも `AlwaysSample()`、`remoteParentNotSampled` / `localParentNotSampled` いずれも `NeverSample()`）では、rootで一度決まったsampled/not-sampledが、

- 同一プロセス内では`context.Context`経由で子spanに、
- プロセスをまたぐ場合はW3C Trace Contextの`traceparent`ヘッダーの`sampled` flag経由でリモートの子spanに、

そのまま継承される。結果として、素直に`ParentBased`をデフォルトのまま使っている限り、トレース全体で一貫した判断になる。

これはOpenTelemetry公式ドキュメントの以下の記述とも一致する。

> Consistent Probability Sampling ... ensures that whole traces are sampled ... at a consistent rate, such as 5% of all traces

W3C Trace Context仕様側でも、`sampled` flagは`trace-flags`フィールドの最下位ビットとして定義されている。

> When set, the least significant bit (right-most), denotes that the caller may have recorded trace data.

この1ビットが`traceparent`ヘッダーで伝播することが、プロセスをまたいだ一貫性を支える仕組みになっている。

### 3. しかし「一部のspanだけがサンプリングされない」ことは仕組み上あり得る

上記2.はあくまで「デフォルト設定のまま素直に使った場合」の結果論であり、以下のようなケースでは同一トレース内で記録の有無が食い違いうる。

**(a) カスタムサンプラーがspan単位の属性で判断を変える場合**
`ShouldSample`にはspan名・`SpanKind`・attributesが渡されるため、例えば「特定のspan名やattributeを持つspanだけをDropする」独自サンプラーを実装すれば、同一トレース内の一部spanだけを記録しない挙動を意図的に作れる。これはSDKのAPI設計上明示的に許容されている（`SamplingParameters`の構造そのものがこれを可能にしている）。

**(b) サービスごとにサンプラー設定が異なり、`ParentBased`を使わない場合**
分散システムでは各サービスが個別にSDK（`TracerProvider`）を初期化する。あるサービスが`ParentBased`を使わずに単体の`TraceIDRatioBased`等を使っていたり、`ParentBased`の`remoteParentSampled`/`remoteParentNotSampled`を独自の値に上書きしていたりすると、上流の判断を無視して独自にDrop/Sampleを決めることになり、トレースの一部区間だけが記録されない状況が起こり得る。

**(c) W3C仕様上も`sampled` flagは「推奨」であり「強制」ではない**
W3C Trace Context仕様は、`sampled` flagについて明確にこう述べている。

> These flags are recommendations given by the caller rather than strict rules to follow ... Because of these issues, tracing vendors make their own recording decisions.

つまり、上流から伝播された`sampled` flagを下流の実装が尊重するかどうかは、仕様上は各実装（vendor/SDK）の裁量に委ねられている。`ParentBased`を使わない、あるいは独自ロジックで上書きする実装であれば、上流のサンプリング判断を無視することが仕様上「不正」にはならない。

**(d) SDKには`Drop`と`RecordOnly`という中間状態も存在する**
`trace/sampling.go`の`AlwaysRecord`サンプラーは、委譲先のroot samplerが`Drop`と判断した場合でも`RecordOnly`（spanは生成・記録されるが`sampled` flagは立たずexportはされない）に変換する。

```go
func (ar alwaysRecord) ShouldSample(p SamplingParameters) SamplingResult {
    rootSamplerSamplingResult := ar.root.ShouldSample(p)
    if rootSamplerSamplingResult.Decision == Drop {
        return SamplingResult{Decision: RecordOnly, ...}
    }
    return rootSamplerSamplingResult
}
```

これは「span-to-metrics処理のために全spanをプロセッサに見せたいが、exportはしたくない」用途向けの機能で、「記録される／されない」と「exportされサンプル済み扱いになる／ならない」が別軸であることを示している。

以上から、**samplingは「トレースIDに基づく一貫した判断」を志向した仕組みではあるが、それを保証するのは各実装・各サービスの設定（`ParentBased`を使うこと、上流の`sampled` flagを尊重すること）であり、仕組みそのものが強制しているわけではない**。設定次第では同一トレース内の一部のspanだけがサンプリングされない状況は起こり得る。

### 4. 本リポジトリの検証環境（`tempo-split-trace`）との関係

[`tempo-split-trace-report.md`](./tempo-split-trace-report.md)の検証環境（`experiment/tempo-split-trace/service-a/main.go`、`service-b/main.go`）を確認すると、両サービスとも`sdktrace.NewTracerProvider(...)`の呼び出しで`WithSampler`を指定しておらず、`docker-compose.yaml`等にも`OTEL_TRACES_SAMPLER`環境変数は設定されていない。したがって両サービスとも**デフォルトの`ParentBased(AlwaysSample())`**で動作しており、常に100%のspanが`RecordAndSample`される。

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(time.Second)),
    sdktrace.WithResource(res),
)
```

（`WithSampler`が渡されていない。`service-a/main.go`・`service-b/main.go`とも同様。）

よって、[`tempo-split-trace-report.md`](./tempo-split-trace-report.md)で観測された「トレースが分割される」現象は、**samplingによってspanが欠落したものではない**。あの検証ではservice-a・service-bとも生成した全spanを100%記録・exportしており、その上で**export先（tempo1 / tempo2という異なるバックエンド）へのルーティングが分岐していた**、という別レイヤーの問題である。

まとめると、

| 現象 | 発生レイヤー | 本リポジトリでの実例 |
| --- | --- | --- |
| 一部のspanだけがサンプリングされない（記録・export自体がされない） | Sampler（`ShouldSample`の判断） | 未検証・今回のexperiment環境では発生しない（常時100%サンプル） |
| 記録・export済みのspanが、別々のバックエンドに送られる | Exporter/Collectorのルーティング設定 | [`tempo-split-trace-report.md`](./tempo-split-trace-report.md)で検証済み |

この2つは原因も対策も異なる別問題であり、混同しないよう注意が必要である。
