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