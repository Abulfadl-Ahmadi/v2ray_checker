# Agent Guidelines & Execution Environment

## 🚨 EXECUTION ENVIRONMENT & ADAPTIVE SHELL DETECTION

**All development tasks, builds, tests, scripts, and runtime operations MUST target the Linux (WSL2 Ubuntu) environment.**
However, the agent must adaptively detect where its current shell / runner is executing:

### 1. Detection Rule (Where is the agent running?):
- **Inside Linux / WSL2 directly** (e.g. user launched `agy`, Cloud Code, or VS Code inside WSL):
  - The shell is Bash/Zsh and the OS is Linux (`uname -s` returns `Linux`, filesystem root is `/`).
  - **DO NOT prefix commands with `wsl`**. Run all Linux commands, Go builds, Python scripts, Git commands, etc., **directly and natively**:
    ```bash
    go build ./...
    python3 main.py
    apt update
    ```
- **From Windows Host** (e.g. runner is PowerShell / CMD, or `<user_information>` specifies OS is Windows):
  - **DO NOT** execute dev tasks directly on Windows host.
  - **MUST wrap all commands with WSL:**
    ```bash
    wsl -d Ubuntu bash -c "<command>"
    ```

---

## 📂 Paths & Mapping

- **Inside Linux (WSL2):**
  - Workspace: `/mnt/f/v2ray_checker` (or `~/v2ray_checker`)
  - Project Directory: `/mnt/f/v2ray_checker/V2rayCollector`
  - Linux Home: `/home/abulfadl` (`~`)
  - Windows User Home: `/mnt/c/Users/Mester` (shortcut `~/win`)
  - Mounted Windows Drives: `/mnt/c`, `/mnt/d`, `/mnt/e`, `/mnt/f` (shortcuts in `~`: `~/e`, `~/f`, `~/drives`)

- **From Windows Host:**
  - Workspace: `F:\v2ray_checker`
  - Project Directory: `F:\v2ray_checker\V2rayCollector`
  - Windows User Directory: `C:\Users\Mester`

---

## 🛠️ Tooling & Packages

- Always use Linux-based package managers and binaries (`apt`, `python3`, `pip`, `go`, etc.).
- **Antigravity CLI:** Installed inside WSL at `~/.local/bin/agy` and available globally as `agy`.
- Ensure files created or modified use **LF (Linux)** line endings.

---

## 🎯 Architecture & Project Vision

- Refer to `PROJECT_VISION.md` (`/mnt/f/v2ray_checker/PROJECT_VISION.md`) for full technical architecture, system goals, and roadmap.
- The project is evolving into an automated 24/7 aggregator, real-world health checker (via Sing-Box / Xray Core), metadata enricher (IP, Geo, Latency), and REST/Subscription API service for V2Ray configurations.
