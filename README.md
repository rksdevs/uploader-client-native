# WoW Logs Uploader – Native Client

The **WoW Logs Uploader** is the official native desktop client for [wow-logs.co.in](https://wow-logs.co.in), a log analysis and leaderboard platform for _World of Warcraft: Wrath of the Lich King_.

This application is built with **[Wails](https://wails.io)** (Go + React/TypeScript) to provide a seamless and automated log uploading experience, eliminating the need to manually zip and upload `WoWCombatLog.txt` files through a browser.

---

## ✨ Features

- **Persistent Directory** – Save your WoW Logs folder path once and reuse it automatically.
- **Multi-Instance Support** – Detect and process multiple raid instances from a single combat log.
- **Real-time Notifications** – Desktop alerts for upload progress and completion.
- **Cross-Platform** – A single codebase for both Windows and macOS.
- **Downloadable Exe file** – Available for Windows currently, For Mac, I will release soon.

---

## ⚙️ Development Setup

### Prerequisites

Make sure the following are installed:

- [Go](https://go.dev/dl/) (latest)
- [Node.js (LTS)](https://nodejs.org/en/) + npm
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Installation & Running

Clone the repository:

```bash
git clone https://github.com/rksdevs/uploader-client-native.git
cd uploader-client-native
```

Install frontend dependencies:

```bash
cd frontend
npm install
cd ..
```

Run in development mode:

```bash
wails dev
```

---

## 📦 Building for Production

### Windows (.exe)

```bash
wails build -platform windows/amd64
```

Output: `build/bin/wow-logs-native-uploader.exe`

### macOS (.app)

_(requires macOS + Xcode tools)_

```bash
wails build -platform darwin/universal
```

Output: `build/bin/wow-logs-native-uploader.app` (zip before distributing).

---

## 🛠️ Technology Stack

- **Framework**: Wails v2
- **Backend**: Go
- **Frontend**: React + TypeScript + Vite
- **Styling**: CSS

---

## 🤝 Contributing

Contributions are welcome!

1. Fork the repository.
2. Create a feature branch:
   ```bash
   git checkout -b feature/AmazingFeature
   ```
3. Commit your changes:
   ```bash
   git commit -m "Add AmazingFeature"
   ```
4. Push to your branch:
   ```bash
   git push origin feature/AmazingFeature
   ```
5. Open a Pull Request.

---

## 📌 About

This client is the **official companion app** for [wow-logs.co.in](https://wow-logs.co.in).
