# Kubernetes Cluster TUI (Bubbletea)

kube-ops-viewにインスパイアされた、Kubernetesクラスターをターミナル上で可視化するTUIアプリケーションです。
[Bubbletea](https://github.com/charmbracelet/bubbletea)と[Lipgloss](https://github.com/charmbracelet/lipgloss)を使用して構築されています。

## 特徴

### 🎨 ビジュアル
- **kube-ops-view風のグリッドレイアウト**: ノードが横に並び、各ノード内でPodが四角いブロックとして表示
- **コンパクトなデザイン**: 余白を最小限にし、多くの情報を一覧表示
- **リソース使用率バー**: 各ノードにCPU/メモリの使用率を横棒グラフで表示
  - 🟢 緑: 0-60%（正常）
  - 🟠 オレンジ: 60-80%（注意）
  - 🔴 赤: 80-100%（高負荷）
- **固定サイズのノードボックス**: 各ノードは固定サイズ（34x8）でコンパクトに表示
- **安定したノード順序**: ノードは名前順でソートされ、更新時も並び順が変わらない
- **ミニマルデザイン**: 詳細情報はノード選択時のみ表示、通常時はビジュアルに特化
- **視覚的なステータス表示**: Podをシンボル付き色付きボックスで表現
  - 🟢 ■ 緑: Running状態のPod
  - 🔴 ✗ 赤: Failed状態のPod
  - 🟠 ◯ オレンジ: Pending状態のPod
  - ⚫️ ■ グレー: Succeeded状態のPod
- **ノードステータス**: ボーダーの色でノードの状態を表示
  - 緑のボーダー: Ready状態のノード
  - 赤のボーダー: NotReady状態のノード
  - 紫の太いボーダー: 選択中のノード

### 🖱️ インタラクティブ機能
- **ノード選択**: 矢印キー（または h/l）でノードを選択
- **詳細表示**: Enterキーで選択したノードの詳細情報を表示
- **Pod選択**: 詳細表示中に矢印キー（または hjkl）でPodを選択
- **Pod詳細**: 選択したPodの詳細情報（名前、Namespace、Status、Ready状態）を表示
- **リアルタイム更新**: 5秒ごとに自動的にクラスター状態を更新
- **レスポンシブレイアウト**: ターミナルサイズに応じて自動的にレイアウトを調整

## 必要要件

- Go 1.21以上
- kubectlが設定済みで、Kubernetesクラスターにアクセス可能であること
- `~/.kube/config`に有効なkubeconfigが存在すること（またはクラスター内で実行）

## ⚠️ リソース使用率について

現在の実装では、リソース使用率はデモ目的でシミュレーション値（allocatableの60%）を表示しています。
実際のリソース使用率を表示するには、以下の対応が必要です：

1. **Metrics Serverのインストール**
   ```bash
   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
   ```

2. **メトリクスの取得**
   現在は`client-go`の基本APIのみ使用していますが、実際の使用率を表示するには：
   - Metrics Server APIを使用（`kubectl top node`相当）
   - または Prometheus等のモニタリングツールと連携

デフォルトではリソース容量の情報を元にシミュレーション表示していますが、視覚的な確認には十分です。

## インストール

```bash
# 依存関係のインストールとビルド
go mod tidy
go build -o kube-ops-tui .
```

## 使い方

```bash
# 実行
./kube-ops-tui
```

### キーボード操作

#### 基本操作
- `←` / `→` または `h` / `l`: ノードを左右に選択
- `↑` / `↓` または `k` / `j`: Pod選択時に上下に移動
- `Enter` または `Space`: 選択したノードの詳細表示を切り替え
- `Esc`: 選択を解除して全体ビューに戻る
- `r`: 手動でクラスター情報を再取得
- `q` または `Ctrl+C`: アプリケーションを終了

#### 使い方の流れ
1. `→` / `l` でノードを選択
2. `Enter` で詳細パネルを表示
3. `←` / `→` / `↑` / `↓` または `h` / `j` / `k` / `l` でPodを選択
4. 選択したPodの詳細情報が表示される
5. `Esc` で選択解除

## アーキテクチャ

### ファイル構成

- `main.go`: Bubbleテアのメインアプリケーション、TUIのレンダリングとイベント処理
- `k8s.go`: Kubernetesクライアントとデータ取得ロジック

### データフロー

1. アプリケーション起動時にKubernetesクライアントを初期化
2. 5秒ごとに`fetchClusterData`を実行してノードとPod情報を取得
3. 取得したデータを`model`に格納
4. `View()`メソッドで視覚的にレンダリング

### 主要コンポーネント

#### Model (Bubbletea)
```go
type model struct {
    k8sClient   *K8sClient
    nodes       []NodeInfo
    width       int
    height      int
    err         error
    loading     bool
    lastUpdate  time.Time
}
```

#### Kubernetes Data Types
```go
type NodeInfo struct {
    Name      string
    Status    string
    CPUUsage  string
    MemUsage  string
    Pods      []PodInfo
    Ready     bool
}

type PodInfo struct {
    Name      string
    Namespace string
    Status    string
    Ready     bool
    Node      string
}
```

## カスタマイズ

### 更新間隔の変更

`main.go`の`tickCmd()`関数で更新間隔を変更できます：

```go
func tickCmd() tea.Cmd {
    return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}
```

### ノードボックスのサイズ変更

`main.go`の`renderNodeBox()`関数で、ノードボックスのサイズを変更できます：

```go
const nodeBoxWidth = 34   // ボックスの幅（コンパクト）
const nodeBoxHeight = 8   // ボックスの高さ（コンパクト）
```

### Podグリッドのカラム数変更

`main.go`の`renderNodeBox()`関数で、1行に表示するPod数を変更できます：

```go
podsGrid := renderPodsGrid(node.Pods, -1, 8) // 8 pods per row（最後の引数を変更）
```

### Podのシンボル変更

`main.go`の`renderPodBox()`関数でシンボルをカスタマイズできます：

```go
symbol := "■"
if pod.Status == "Running" {
    symbol = "■"  // 実行中
} else if pod.Status == "Pending" {
    symbol = "◯"  // 保留中
} else if pod.Status == "Failed" {
    symbol = "✗"  // 失敗
}
```

### スタイルのカスタマイズ

`main.go`のスタイル定義をカスタマイズできます：

```go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(lipgloss.Color("#7D56F4")).
        Padding(0, 1)

    nodeBoxReadyStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#04B575")).
        Padding(1, 2).
        Width(35)
    // ... その他のスタイル
)
```

## 今後の拡張案

- [x] インタラクティブな操作（ノード・Pod選択、詳細表示）
- [x] kube-ops-viewのような2Dグリッドレイアウト
- [x] ミニマルでビジュアルに特化したデザイン
- [ ] ネームスペースフィルタリング
- [ ] Deployment/Service/ConfigMapなど他のリソースの表示
- [ ] メトリクスサーバーとの連携（実際のCPU/メモリ使用率をグラフ表示）
- [ ] ログビューア（選択したPodのログをリアルタイム表示）
- [ ] イベント表示（Kubernetesイベントストリームの監視）
- [ ] マウスサポート（クリックで選択）
- [ ] 検索機能（Pod名やNamespaceでフィルター）

## 参考

- [kube-ops-view](https://github.com/hjacobs/kube-ops-view)
- [Bubbletea](https://github.com/charmbracelet/bubbletea)
- [Lipgloss](https://github.com/charmbracelet/lipgloss)
- [Kubernetes Client Go](https://github.com/kubernetes/client-go)
