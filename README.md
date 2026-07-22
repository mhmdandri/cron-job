# Shipment Cron Job Service (Go + Gin)

Aplikasi Scheduler Cron-Job ringan dan efisien yang dibangun menggunakan **Go**, **Gin Web Framework**, dan `robfig/cron/v3` untuk secara otomatis melakukan HTTP request ke endpoint monitor shipment.

---

## 🚀 Fitur Utama

- **Target Hit API**: `https://shipment.mohaproject.tech/api/cron/monitor`
- **Authorization**: `Bearer KMASLDKMNOINFNKmknsnqiwjij19980jn`
- **Interval Fleksibel**:
  - Default: **15 menit** (`15m` atau `*/15 * * * *`).
  - Mudah diubah ke **30 menit** (`30m`), **10 menit**, atau format kustom lainnya.
  - Dapat diubah **secara dinamis tanpa perlu restart server** melalui Web UI Dashboard maupun API endpoint `POST /api/schedule`.
- **Manual Trigger**: Tombol / Endpoint untuk melakukan hit API secara instant kapan saja.
- **Live History Logs**: Menyimpan riwayat eksekusi (HTTP Status, Durasi ms, dan Response Body).
- **Web UI Dashboard**: Antarmuka berbasis web modern & responsif di `http://localhost:8080`.
- **Deployment Ready**: Dilengkapi dengan `Dockerfile`, `docker-compose.yml`, dan `cron-job.service` (Systemd).

---

## 🛠️ Cara Mengakses Web UI Dashboard & API Endpoints

| Endpoint | Method | Deskripsi |
|---|---|---|
| `/` | `GET` | Dashboard UI Interaktif |
| `/health` | `GET` | Health Check Service |
| `/api/status` | `GET` | Info Schedule saat ini, next run, & statistik |
| `/api/logs` | `GET` | Riwayat log hasil hit API |
| `/api/trigger` | `POST` | Hit API secara manual |
| `/api/schedule` | `POST` | Ubah interval secara dinamis (`{"interval": "30m"}`) |

---

## 💻 Jalankan Lokal (Development)

1. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

2. **Jalankan Service**:
   ```bash
   go run main.go
   ```

3. **Buka di Browser**:
   Buka `http://localhost:8080` untuk melihat Dashboard UI.

---

## 🌐 Cara Deploy di Linux VPS

### Opsi A: Menggunakan Docker & Docker Compose (Rekomendasi)

1. Clone / Copy repository ini ke VPS.
2. Jalankan perintah berikut di VPS:
   ```bash
   docker compose up -d --build
   ```
3. Cek status container:
   ```bash
   docker compose ps
   ```

---

### Opsi B: Deploy Menggunakan Systemd (Direct Binary)

1. **Build Binary untuk Linux di VPS (atau Cross-Compile)**:
   ```bash
   CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o cron-job main.go
   ```

2. **Pindahkan File ke Directory VPS**:
   ```bash
   mkdir -p /var/www/cron-job
   cp cron-job /var/www/cron-job/
   cp -r static /var/www/cron-job/
   cp .env /var/www/cron-job/
   ```

3. **Install Service Systemd**:
   ```bash
   cp cron-job.service /etc/systemd/system/
   systemctl daemon-reload
   systemctl enable cron-job
   systemctl start cron-job
   ```

4. **Cek Status Service**:
   ```bash
   systemctl status cron-job
   journalctl -u cron-job -f
   ```

---

## ⚙️ Konfigurasi Environment (`.env`)

```env
PORT=8080
TARGET_URL=https://shipment.mohaproject.tech/api/cron/monitor
BEARER_TOKEN=KMASLDKMNOINFNKmknsnqiwjij19980jn
CRON_SCHEDULE=15m
```
