# 設定

このドキュメントでは、CronWorkflow Replicatorツールの設定方法について説明します。

## パス解決

このツールは、設定ファイル内の相対パスを現在の作業ディレクトリではなく、設定ファイルの場所を基準として解決します。これにより、コマンドを実行する場所に関係なく一貫した動作が保証されます。

### サポートされるパスフィールド

- `outputDirectory`: 生成されたYAMLファイルの出力ディレクトリ
- `baseManifestPath`: ベースCronWorkflowマニフェストテンプレートへのパス

### パス解決の動作

ツールは以下の方法でパス解決を処理します：

- **相対パス**: 現在の作業ディレクトリではなく、常に設定ファイルのディレクトリを基準に解決
- **絶対パス**: 変更されることなくそのまま使用
- **ネストしたパス**: 相対パスと絶対パスの両方で正しく動作

### パスの例

```yaml
units:
  - outputDirectory: "./output"              # 設定ファイルを基準に解決
    baseManifestPath: "./base-manifest.yaml" # 設定ファイルを基準に解決
    # ...
  - outputDirectory: "manifests/output"      # ネストした相対パス
    baseManifestPath: "templates/base.yaml"  # ネストした相対パス
    # ...
  - outputDirectory: "/absolute/path/output"         # 絶対パスはそのまま
    baseManifestPath: "/absolute/path/base.yaml"     # 絶対パスはそのまま
    # ...
```

### 異なるディレクトリからの実行

つまり、どのディレクトリからでもツールを実行でき、正しく動作します：

```bash
# これらはすべて同じように動作します：
./cron-workflow-replicator --config examples/v1alpha1/basemanifest/config.yaml
cd /tmp && /path/to/cron-workflow-replicator --config /path/to/examples/v1alpha1/basemanifest/config.yaml
```

### パス解決の特殊なケース

- 設定ファイルが `/project/configs/app.yaml` にあり、`outputDirectory: "./output"` と指定した場合、ファイルはコマンドを実行した場所の `./output` ではなく、`/project/configs/output/` に書き込まれます
- この動作は `outputDirectory` と `baseManifestPath` フィールドの両方に一貫して適用されます

## Kustomize統合

ツールは、CronWorkflowマニフェストを生成する際にKustomizeのkustomization.yamlファイルを自動的に管理できます。

### Kustomize統合の有効化

ユニットにkustomize設定を追加します：

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    kustomize:
      update-resources: true
    # ... 残りの設定
```

### Kustomize統合の動作

`kustomize.update-resources: true` が設定されている場合：

1. ツールは指定された出力ディレクトリにCronWorkflow YAMLファイルを生成
2. 同じディレクトリに `kustomization.yaml` ファイルを自動的に作成または更新
3. `kustomization.yaml` の `resources` リストに生成されたすべてのファイルを含める
4. kustomization更新が失敗した場合、警告をログに出力するが、他のファイルの処理は続行

### 例

ユニットが `workflow-1.yaml`、`workflow-2.yaml` のようなファイルを生成する場合、ツールは以下を作成します：

```yaml
# kustomization.yaml（自動生成）
resources:
- workflow-1.yaml
- workflow-2.yaml
```

## ファイル名と衝突処理

設定内の複数の値が同じ名前のファイルを生成する場合、ツールは数字のサフィックスを追加することで衝突を自動的に処理します。

### 命名の動作

- 最初のファイル: `filename.yaml`
- 同じ名前の2番目のファイル: `filename-2.yaml`
- 同じ名前の3番目のファイル: `filename-3.yaml`
- 以下同様...

### 例

設定がすべて `daily-job.yaml` という名前になる複数のワークフローを生成する場合、以下のようになります：
- `daily-job.yaml`
- `daily-job-2.yaml`
- `daily-job-3.yaml`

これにより、ファイルが上書きされることなく、生成されたすべてのマニフェストが保持されます。

## 設定ファイル構造

設定ファイルは、CronWorkflowをどのように生成するかを定義します。設定内の各 `unit` は、作成されるCronWorkflowのセットを表します。

### 基本設定

```yaml
units:
  - outputDirectory: "./output"
    # 基本的なunit設定
```

### ベースマニフェスト付き

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    # ベーステンプレートを使用したunit設定
```

### カスタム値付き

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./base-manifest.yaml"
    # カスタム値をテンプレートに注入可能
```

## 例

完全な設定例については `examples/` ディレクトリを確認してください：

- `examples/v1alpha1/novalue/` - カスタム値なしの基本設定
- `examples/v1alpha1/withvalue/` - カスタム値ありの設定
- `examples/v1alpha1/basemanifest/` - ベースマニフェストテンプレートを使用した設定
- `examples/v1alpha1/kustomize/` - Kustomize統合を有効にした設定