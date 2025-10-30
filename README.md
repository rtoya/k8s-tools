# k8s-tools

Kubernetes運用のための便利ツール群

## 📦 含まれるツール

### [kube-view](./kube-view/)

kube-ops-viewにインスパイアされた、Kubernetesクラスター可視化TUIツール

**主な機能:**
- 📊 グリッドレイアウトでノードとPodを視覚的に表示
- 📈 CPU/メモリ使用率を色分けバーで表示
- 🎮 インタラクティブなノード/Pod選択
- 🔄 リアルタイム自動更新（5秒間隔）
- 🎨 コンパクトで見やすいデザイン

```bash
cd kube-view
go build -o kube-ops-tui .
./kube-ops-tui
```

詳細は [kube-view/README.md](./kube-view/README.md) を参照してください。

## 🚀 必要要件

- Go 1.21以上
- kubectl設定済み（`~/.kube/config`）
- Kubernetesクラスターへのアクセス権

## 📋 今後の予定

以下のツールを追加予定：

- [ ] **kube-exec**: 複数Podへの一括コマンド実行
- [ ] **kube-logs**: 複数Podのログを集約表示
- [ ] **kube-switch**: Kubernetesコンテキスト/ネームスペースの高速切り替え
- [ ] **kube-resource**: リソース使用状況の詳細分析
- [ ] **kube-events**: Kubernetesイベントのリアルタイムモニタリング

## 🛠️ 開発

各ツールは独立したディレクトリで管理されています：

```
k8s-tools/
├── README.md           # このファイル
├── kube-view/          # クラスター可視化TUI
│   ├── main.go
│   ├── k8s.go
│   └── README.md
├── kube-exec/          # (予定)
├── kube-logs/          # (予定)
└── ...
```

### 各ツールのビルド

```bash
cd <tool-name>
go build -o <binary-name> .
```

### すべてのツールをビルド

```bash
# ビルドスクリプト作成予定
./build-all.sh
```

## 🔗 参考

- [kube-ops-view](https://github.com/hjacobs/kube-ops-view) - インスピレーション元
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUIフレームワーク
- [Kubernetes Client Go](https://github.com/kubernetes/client-go) - Kubernetes API クライアント
