# V2Ray Collector & Automated 24/7 Health Checker Service

## 📌 Project Overview
This project evolves the base **V2rayCollector** into a complete, automated, 24/7 proxy ecosystem. It aggregates V2Ray/Xray proxy configurations from various Telegram channels/sources, tests and validates their real-world connectivity using **Sing-Box Core**, extracts network/geographic metadata, stores active nodes in **SQLite**, and exposes them via a high-performance REST and Subscription API, packaged with **Docker Compose** for seamless VPS deployment.

---

## 🎯 Architectural Decisions & Specifications

1. **Testing Engine (Checker Core)**:
   - **Sing-Box Core:** High performance, low memory footprint, native support for modern protocols (VLESS Reality, VMess, Trojan, Shadowsocks, Hysteria 2, TUIC).
   - **True Delay (RTT) Verification:** Real HTTP/HTTPS handshake to configurable probe endpoints (e.g. Cloudflare trace, Google 204).

2. **Storage & State Management**:
   - **SQLite Database:** Stores live active configs, historical latency, uptime percentage, first seen, and last checked timestamps.
   - Automatically prunes dead configurations after consecutive failures.

3. **24/7 Scheduler & Worker Engine**:
   - Background worker with interval configurable via config.yaml / environment variables (default: 15 minutes).
   - Concurrency control for fast parallel checking without overwhelming CPU or triggering rate limits.

4. **REST API & Subscription Endpoints**:
   - **JSON API (/api/nodes)**:
     `json
     [
       {
         config: vless://...,
         ping: 62,
         country: IR,
         ip: 185.243.45.15
       }
     ]
     `
   - **Subscription Feeds**:
     - Raw Base64 subscription (/sub/all, /sub/vless, /sub/vmess, etc.) for direct import into v2rayNG, Streisand, V2Box, etc.
     - Clash Meta / Mihomo YAML subscription (/sub/clash).
     - Sing-box JSON subscription (/sub/sing-box).

5. **Deployment & Packaging**:
   - Dockerfile + docker-compose.yml for 1-click 24/7 VPS setup with persistent SQLite storage volume.
   - Clean configuration via config.yaml.

---

## 💡 Future Enhancements

- **Operator Compatibility Tagging:** Tag proxies according to compatibility with Iranian mobile operators (MCI, Irancell, Rightel, Shatel).
- **Auto-rotation Local Proxy Pool:** Expose a local SOCKS5/HTTP forwarder routing traffic to the healthiest available node.
- **Web Dashboard:** Minimal web UI for monitoring node counts, status, and copying subscription links.
