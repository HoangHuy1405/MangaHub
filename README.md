# MangaHub

MangaHub is a comprehensive manga reading and management platform. The project is organized into two main parts to cleanly separate the frontend interface from the backend system.

## Project Structure

- **`/client`**: The frontend application. This directory will contain the UI components, state management, and styling.
- **`/server`**: The backend application built with Go. It serves as the core API, handling business logic, data storage, and external integrations (REST API, WebSockets, etc.).

## Architecture Highlights

The backend (`/server`) is designed around a robust 3-Layer Architecture:
1. **Handler Layer**: Processes incoming HTTP/TCP/WebSocket requests, validates payloads, and formats the API responses.
2. **Service Layer**: Orchestrates the core business logic and rules of the application.
3. **Repository Layer**: Manages data persistence and database interactions (PostgreSQL/SQLite).

## Getting Started

### Backend
1. Navigate to the server directory: `cd server`
2. Run the API Server: `go run cmd/api-server/main.go`

### Frontend
*(Frontend setup instructions will be added once the client application is initialized)*

---
*NetCentric Project - MangaHub*
