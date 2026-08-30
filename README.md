# ⚡ V2Ray Proxy Aggregator & 24/7 Automated Health Checker Service

[![Build & Release Binaries](https://github.com/Abulfadl-Ahmadi/v2ray_checker/actions/workflows/release.yml/badge.svg)](https://github.com/Abulfadl-Ahmadi/v2ray_checker/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

یک سرویس ۲۴/۷ خودکار برای جمع‌آوری کانفیگ‌های V2Ray (VLESS, VMess, Trojan, Shadowsocks)، تست زنده سلامت و پینگ واقعی نودها، ذخیره در SQLite و ارائه REST JSON API و لینک سابسکریپشن کلاینت‌های موبایل (v2rayNG, Streisand, V2Box).

---

## 🚀 نصب و اجرای تک‌خطی (One-Click Installation)

برای نصب خودکار فایل باینری کامپایل‌شده از صفحه ریلیزها و تنظیم سرویس دائمی ۲۴ ساعته (Systemd) روی هر سرور اوبونتو/دبیان، دستور زیر را در ترمینال سرور اجرا کنید:

```bash
curl -fsSL https://raw.githubusercontent.com/Abulfadl-Ahmadi/v2ray_checker/main/install.sh | sudo bash
```

> **نکته:** این اسکریپت به‌طور خودکار معماری سرور (x86_64 یا ARM64) را تشخیص داده، آخرین باینری را از گیت‌هاب دانلود کرده، فایل `config.yaml` را تنظیم و سرویس `systemd` را به صورت دائمی فعال می‌کند.

---

## 📡 اندپوینت‌های API و سابسکریپشن

پس از اجرا، وب‌سرور روی پورت `8080` در دسترس خواهد بود:

### ۱. اندپوینت JSON با فرمت درخواستی (`GET /api/nodes`)

```bash
curl http://YOUR_SERVER_IP:8080/api/nodes
```

نمونه خروجی:
```json
[
  {
    "config": "vless://2e622f08-444a-46d0-8dac-6f9709f4d172@check-host.net:8880?encryption=none&type=httpupgrade&path=%2F&host=sni.abilim.info",
    "ping": "369",
    "country": "UN",
    "ip": "104.21.74.214"
  },
  {
    "config": "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp5dHcyYXdu@54.36.174.140:443",
    "ping": "398",
    "country": "PL",
    "ip": "54.36.174.140"
  }
]
```

### ۲. لینک سابسکریپشن کلاینت موبایل (`GET /sub/all`)

این لینک را مستقیماً در برنامه‌های **v2rayNG**, **Streisand**, **V2Box** وارد کنید:
```
http://YOUR_SERVER_IP:8080/sub/all
```
- خروجی شامل تنها نودهای ۱۰۰٪ تست‌شده، سالم و فعال به صورت Base64 استاندارد است.

### ۳. آمار وضعیت سرور (`GET /api/stats`)
```bash
curl http://YOUR_SERVER_IP:8080/api/stats
```
خروجی:
```json
{
  "alive_nodes": 347,
  "total_nodes": 414
}
```

---

## 🛠️ دستورات مدیریت سرویس (Systemd Commands)

```bash
# مشاهده وضعیت سرویس
sudo systemctl status v2ray_checker

# مشاهده لاگ‌های زنده
sudo journalctl -u v2ray_checker -f

# ری‌استارت سرویس
sudo systemctl restart v2ray_checker

# متوقف کردن سرویس
sudo systemctl stop v2ray_checker
```

---

## 🐳 روش جایگزین: اجرا با داکر (Docker Compose)

```bash
git clone https://github.com/Abulfadl-Ahmadi/v2ray_checker.git
cd v2ray_checker
docker compose up -d --build
```

---

## ⚙️ پیکربندی (`config.yaml`)

مسیر فایل تنظیمات: `/opt/v2ray_checker/config.yaml`

```yaml
server:
  port: ":8080"

worker:
  check_interval: 15m
  concurrency: 30
  timeout_sec: 4
  max_failures: 3

probe:
  urls:
    - "https://1.1.1.1/cdn-cgi/trace"
    - "https://www.google.com/generate_204"

database:
  path: "./data/v2ray.db"

collector:
  channels_file: "./channels.csv"
  subscription_urls:
    - "https://raw.githubusercontent.com/Abulfadl-Ahmadi/V2rayCollector/refs/heads/main/mixed_iran.txt"
```
