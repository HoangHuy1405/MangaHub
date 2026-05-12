# MangaHub

MangaHub is a comprehensive manga tracking, reading, and management platform. It features a robust multi-server architecture that supports HTTP API requests, real-time TCP synchronization, UDP notifications, gRPC internal services, and WebSocket-based chat.

## 🌟 Features
- **Manga Discovery & Library:** Browse mangas and manage your personal library.
- **Reading Progress Tracking:** Track chapters and volumes read.
- **Real-Time Sync:** Synchronize reading progress across devices over TCP.
- **Live Notifications:** Receive real-time system and update notifications via UDP.
- **Interactive Chat:** Join WebSocket-based chat rooms to discuss mangas.
- **Robust Multi-Server Architecture:** HTTP, TCP, UDP, gRPC, and WebSocket servers seamlessly working together.

## 🛠️ Prerequisites

Before you start, ensure you have the following installed on your machine:
- **[Go](https://golang.org/doc/install)** (1.20 or later)
- **Git**

## 🚀 Getting Started

Follow these steps to set up the MangaHub project from scratch:

### 1. Clone the Repository

```bash
git clone https://github.com/HoangHuy1405/MangaHub.git
cd MangaHub
```

### 2. Configure the Environment (Optional)
The server uses SQLite by default, so no complex database setup is required. However, you can configure ports and secrets if needed:
```bash
cd server
cp .env.example .env
```

### 3. Build and Install the CLI
MangaHub provides a powerful Command-Line Interface (CLI) to interact with the servers and manage your data.
You can install the CLI globally using `go install`:

```bash
cd server
go mod download
go install ./cmd/mangahub
```
> **Note**: Ensure your Go binary path (`$GOPATH/bin` or `%USERPROFILE%\go\bin`) is added to your system's PATH. Alternatively, you can build the executable directly: `go build -o mangahub.exe ./cmd/mangahub` and run `./mangahub.exe`.

### 4. Start the MangaHub Servers
Once the CLI is installed, you can start all backend services with a single command:

```bash
mangahub server start
```
This command will automatically compile and launch the following components in the background:
- HTTP API Server (`localhost:8080`)
- TCP Sync Server (`localhost:9090`)
- UDP Notification Server (`localhost:9091`)
- gRPC Internal Service (`localhost:9092`)
- WebSocket Chat Server (`localhost:9093`)

You can check the status of your servers at any time:
```bash
mangahub server status
```

To stop the servers safely:
```bash
mangahub server stop
```

---

## 💻 CLI Usage Guide

With the servers running, you can use the `mangahub` CLI to interact with the platform. Here are some of the most common commands:

### Authentication
Create an account and log in to get your session token:
```bash
mangahub auth register --username <name> --email <email> --password <pass>
mangahub auth login --username <name> --password <pass>
```

### Manga & Library
Browse mangas and manage your collection:
```bash
mangahub manga list
mangahub manga search --query "Naruto"
mangahub library add --id <manga_id>
mangahub library list
```

### Progress Tracking & Sync
Update your reading progress and sync it across devices:
```bash
mangahub progress update --manga-id <manga_id> --chapter 5
mangahub progress history
mangahub sync start
```

### Interactive Features
Join chat rooms or listen to live notifications:
```bash
mangahub chat join --room general
mangahub notify listen
```

### Statistics & Management
View stats and export your library data:
```bash
mangahub stats view
mangahub export library
```

To see all available commands and detailed usage, you can always run:
```bash
mangahub --help
```

## 🏗️ Architecture Highlights

The backend (`/server`) is designed around a robust 3-Layer Architecture:
1. **Handler Layer**: Processes incoming HTTP/TCP/WebSocket requests, validates payloads, and formats API responses.
2. **Service Layer**: Orchestrates the core business logic and rules of the application.
3. **Repository Layer**: Manages data persistence and database interactions (SQLite by default).

---
*NetCentric Project - MangaHub*
