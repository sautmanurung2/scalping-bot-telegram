# 🚀 Panduan Deployment Docker Compose: Standalone Telegram Trading Bot

Dokumen ini berisi panduan langkah-demi-langkah deployment aplikasi **Financial Market Analysis Telegram Bot** menggunakan **Docker & Docker Compose**. 

Dengan Docker Compose, bot Anda berjalan secara terisolasi di dalam container produksi, otomatis berjalan di background 24/7, dan otomatis hidup kembali (*auto-restart*) jika terjadi kebocoran jaringan atau reboot pada server.

---

## 📌 1. Prasyarat Deployment (Prerequisites)

Sebelum memulai deployment, pastikan server/komputer target memenuhi prasyarat berikut:

1. **Docker & Docker Compose**: Sudah ter-install di server (VPS Linux / Local).
   ```bash
   # Cek versi Docker & Docker Compose
   docker --version
   docker compose version
   ```
2. **Telegram Bot Token**: Token rahasia dari `@BotFather` di Telegram.

---

## 🛠️ 2. Langkah-Langkah Deployment (Docker Compose)

### Langkah 1: Clone Repository Proyek
```bash
# 1. Clone repository proyek Anda ke server
git clone https://github.com/USERNAME/project-iseng.git /opt/trading-bot
cd /opt/trading-bot
```

### Langkah 2: Buat File `.env` untuk Token Telegram
Buat file `.env` di root folder proyek untuk menyimpan token bot Anda secara aman:

```bash
# Salin dari file contoh
cp .env.example .env

# Isi token bot Telegram Anda
nano .env
```
*Isi file `.env`:*
```env
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
```

### Langkah 3: Build & Jalankan Container
Jalankan perintah berikut untuk mengompresi image dan menyalakan daemon container di background:

```bash
# Build image dan jalankan container di background
docker compose up -d --build
```

---

## 📊 3. Perintah Manajemen Docker Compose

Berikut adalah daftar perintah praktis untuk mengelola bot yang berjalan:

| Perintah | Deskripsi Fungsi |
| :--- | :--- |
| `docker compose ps` | Memeriksa status container bot (apakah *Up/Running*). |
| `docker compose logs -f` | Melihat log aktivitas bot secara *live/real-time*. |
| `docker compose restart` | Merestart ulang container bot. |
| `docker compose stop` | Menghentikan sementara container bot. |
| `docker compose down` | Menghentikan dan menghapus container bot. |

---

## 🔄 4. Cara Update Pembaruan Kode (Auto Update Workflow)

Jika ada perbaikan atau fitur baru dari repository Git:

```bash
# 1. Tarik pembaruan kode dari repository
git pull origin main

# 2. Build ulang container dan restart secara seamless
docker compose up -d --build
```

---

## 💾 5. Database Persistence (SQLite)

File database SQLite `trading_bot.db` telah dipetakan ke *persistent volume* pada `docker-compose.yml`:
```yaml
volumes:
  - ./trading_bot.db:/app/trading_bot.db
```
Sehingga seluruh riwayat sinyal trading yang telah dicatat bot **tidak akan hilang** meskipun container di-build ulang atau di-restart.

---

*Selamat! Telegram Bot Analisis Pasar Keuangan Anda kini resmi beroperasi 24/7 menggunakan Docker Compose.* 🚀
