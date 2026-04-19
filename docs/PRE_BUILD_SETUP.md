# 預編譯環境設定指南

本文件說明如何在 rustdesk-api 所在的機器上設定預編譯 RustDesk Client 所需的環境。

預編譯功能讓你從網頁觸發編譯，產出可供自訂客戶端下載的 base binary。

> 參考來源：`rustdesk/README.md`、`rustdesk/Dockerfile`、`.github/workflows/bridge.yml`、`.github/workflows/flutter-build.yml`

---

## 1. 系統套件（Ubuntu/Debian）

```bash
sudo apt update -y && sudo apt install -y \
    g++ gcc git curl wget nasm yasm make unzip zip sudo ca-certificates \
    clang libclang-dev llvm-dev \
    cmake ninja-build pkg-config \
    libgtk-3-dev \
    libxcb-randr0-dev libxdo-dev libxfixes-dev libxcb-shape0-dev libxcb-xfixes0-dev \
    libasound2-dev libpulse-dev libpam0g-dev \
    libssl-dev \
    libgstreamer1.0-dev libgstreamer-plugins-base1.0-dev \
    dpkg-dev
```

---

## 2. Rust 工具鏈

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source $HOME/.cargo/env
rustup default 1.75.0
```

> 版本參考：CI 使用 Rust 1.75

---

## 3. Flutter SDK

```bash
# 方法一：使用 snap
sudo snap install flutter --classic

# 方法二：手動安裝
git clone https://github.com/flutter/flutter.git -b 3.24.5 --depth 1 $HOME/flutter
export PATH="$HOME/flutter/bin:$PATH"
flutter precache --linux
```

> 版本參考：CI 使用 Flutter 3.24.5

確認安裝：
```bash
flutter --version
flutter doctor
```

---

## 4. vcpkg（C++ 依賴）

```bash
git clone --branch 2023.04.15 --depth=1 https://github.com/microsoft/vcpkg $HOME/vcpkg
$HOME/vcpkg/bootstrap-vcpkg.sh -disableMetrics
$HOME/vcpkg/vcpkg --disable-metrics install libvpx libyuv opus aom
```

設定環境變數（加到 `~/.bashrc` 或 `~/.profile`）：
```bash
export VCPKG_ROOT=$HOME/vcpkg
```

---

## 5. Flutter Rust Bridge 程式碼產生器

```bash
cargo install flutter_rust_bridge_codegen --version 1.80.1 --features "uuid" --locked
```

> 版本參考：`.github/workflows/bridge.yml`
> 注意：CI 裡還裝了 `cargo-expand`，但實測不裝也能成功編譯。

---

## 6. rustdesk-api 設定（conf/config.yaml）

```yaml
custom-client:
  rustdesk-src-dir: "../rustdesk"  # rustdesk/ 原始碼路徑（相對或絕對）
  build-worktree-dir: "./data/build-worktree"
  build-log-dir: "./data/build-logs"
  base-binaries-dir: "./data/base-binaries"
  cache-dir: "./data/custom-client-cache"
```

確認 `rustdesk-src-dir` 指向的目錄：
- 是一個 git repo
- 有 `Cargo.toml`、`build.py`、`flutter/` 等
- submodule 已初始化（`git submodule update --init --recursive`）

---

## 7. Ed25519 簽章金鑰（已硬編碼，無需設定）

custom.txt 的簽章金鑰已硬編碼在程式碼裡，不需要額外設定：

| 位置 | 值 | 用途 |
|------|-----|------|
| `rustdesk-api/internal/service/custom_client.go` | Private key | API server 用來簽章 custom.txt |
| `rustdesk/src/common.rs:2187` | Public key | Client 端用來驗證 custom.txt |

預編譯時也會自動把 public key patch 到 worktree 的 `common.rs`。

> 如果需要更換金鑰，需同時修改這兩個位置並重新編譯 client。

---

## 8. 驗證環境

以下指令應全部成功：

```bash
cargo --version          # >= 1.75
flutter --version        # 3.24.5
echo $VCPKG_ROOT         # 應指向 vcpkg 目錄
flutter_rust_bridge_codegen --version  # 1.80.1
dpkg-deb --version       # 用於 .deb 打包
```

---

## 環境就緒後

1. 啟動 rustdesk-api：`go run cmd/apimain.go`
2. 開啟 console-web → 自訂客戶端 → 預編譯任務
3. 選擇版本、平台、架構 → 開始編譯
4. 編譯完成後，在「客戶端生成」頁面選擇平台/版本/格式即可下載帶有 custom.txt 的安裝包
