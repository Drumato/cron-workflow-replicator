# 使用方法

このガイドでは、CronWorkflow Replicatorツールの使用方法を説明します。

## 基本的な使用方法

```bash
./cron-workflow-replicator --config path/to/config.yaml
```

## Dockerを使用した実行

Goのインストールやローカルでのバイナリビルドなしに、事前ビルドされたDockerイメージを使用してCLIを実行できます。

### 利用可能なイメージ

- `ghcr.io/drumato/cron-workflow-replicator:main` (amd64)
- `ghcr.io/drumato/cron-workflow-replicator:main-arm` (arm64)

### Dockerでの実行

一時的なDockerコンテナでCLIを実行するには：

```bash
# amd64システム用
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml

# arm64システム用（Apple Siliconなど）
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main-arm \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

`--rm` フラグは実行後にコンテナを自動的に削除し、`-v $(pwd):/workspace -w /workspace` は現在のディレクトリをコンテナ内の作業ディレクトリとしてマウントします。

## 例

### 基本例（値なし）

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

**動作**: 最小限の設定で基本的なCronWorkflow YAMLファイルを生成します（`./output/` に出力）

### 値ありの例

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/withvalue/config.yaml
```

**動作**: 生成されたマニフェストにカスタム値を注入する方法を示します（`./output/` に出力）

### ベースマニフェストを使用した例

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/basemanifest/config.yaml
```

**動作**: ベースマニフェストテンプレートを使用し、異なる設定を適用して複数のバリエーションを作成します（`examples/v1alpha1/basemanifest/output/` に出力）

### Kustomize統合を使用した例

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/kustomize/config.yaml
```

**動作**: CronWorkflowマニフェストを生成し、生成されたすべてのリソースを含むkustomization.yamlファイルを自動的に作成/更新します（`examples/v1alpha1/kustomize/output/` に出力）

## JSONPathを使った設定例

### 例1: 本番環境バックアップワークフロー

```yaml
units:
  - outputDirectory: "./output"
    values:
      - filename: "production-backup"
        paths:
          - path: "$.metadata.name"
            value: "production-daily-backup"
          - path: "$.metadata.namespace"
            value: "production"
          - path: "$.metadata.labels.app"
            value: "backup-service"
          - path: "$.metadata.labels.environment"
            value: "production"
          - path: "$.spec.schedule"
            value: "0 2 * * *"  # 毎日午前2時
          - path: "$.spec.concurrencyPolicy"
            value: "Forbid"
          - path: "$.spec.successfulJobsHistoryLimit"
            value: "3"
          - path: "$.spec.failedJobsHistoryLimit"
            value: "1"
```

### 例2: マルチ環境データ処理

```yaml
units:
  - outputDirectory: "./output"
    values:
      - filename: "data-processing-staging"
        paths:
          - path: "$.metadata.name"
            value: "data-processing-staging"
          - path: "$.metadata.namespace"
            value: "staging"
          - path: "$.spec.schedule"
            value: "0 */4 * * *"  # 4時間おき
          - path: "$.spec.workflowSpec.arguments.parameters[0].value"
            value: "s3://staging-data-bucket/"
      - filename: "data-processing-production"
        paths:
          - path: "$.metadata.name"
            value: "data-processing-production"
          - path: "$.metadata.namespace"
            value: "production"
          - path: "$.spec.schedule"
            value: "0 1 * * *"   # 毎日午前1時
          - path: "$.spec.workflowSpec.arguments.parameters[0].value"
            value: "s3://production-data-bucket/"
```

### 例3: 複雑なネスト設定

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./templates/complex-workflow.yaml"
    values:
      - filename: "ml-training-pipeline"
        paths:
          - path: "$.metadata.name"
            value: "weekly-ml-training"
          - path: "$.spec.schedule"
            value: "0 0 * * 0"  # 毎週日曜日
          - path: "$.spec.workflowSpec.templates[0].container.env[0].value"
            value: "production"
          - path: "$.spec.workflowSpec.templates[0].container.resources.requests.memory"
            value: "8Gi"
          - path: "$.spec.workflowSpec.templates[0].container.resources.requests.cpu"
            value: "4"
          - path: "$.spec.workflowSpec.arguments.parameters[0].name"
            value: "model-version"
          - path: "$.spec.workflowSpec.arguments.parameters[0].value"
            value: "v2.1.0"
```

これらの例では以下を示しています：
- **環境固有設定**: ステージング環境と本番環境での異なるネームスペース、スケジュール、パラメータ
- **リソース管理**: メモリとCPUリクエストの設定
- **複雑なパスターゲティング**: コンテナ環境変数や配列要素など深くネストしたフィールドへのアクセス
- **パラメータ注入**: ワークフローの引数とパラメータの動的設定

## ソースからのビルド

ローカルでバイナリをビルドする場合：

```bash
make build
./bin/cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

## テスト

テストを実行するには：

```bash
make test
```