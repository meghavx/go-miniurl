# Go MiniURL
A lightweight URL shortener optimized for backend performance with rate limiting, caching, and asynchronous click analytics.


#### Live Demo: [go-miniurl.onrender.com](https://go-miniurl.onrender.com)


## Key Highlights

- **Rate-limited** API to prevent abuse
- Redis-based **caching** for fast URL resolution
- **Asynchronous click analytics** to avoid blocking redirects
- **Persistent storage** as the source of truth
- **Bloom filter** for fast existence checks
- Minimal web UI for interaction


## Tech Stack

- **Backend**: Go  
- **Caching, Rate Limiting & Async Analytics**: Redis  
- **Persistence**: SQLite  
- **Frontend**: Server-rendered HTML (HTMX) + Tailwind CSS


## Architecture Overview (HLD)
```
Client
  |
  | HTTP Request
  v
API Layer (Go)
  ├── Rate Limiter (Redis)
  ├── URL Resolver
  |
  ├── Bloom Filter
  |     └── Fast existence check for short codes
  |
  ├── Cache (Redis)
  |     └── ShortURL → LongURL
  |
  ├── Persistent Store (SQLite)
  |     └── URL mappings (source of truth)
  |
  └── Analytics Publisher (Async)
        └── Click events
```

## Run with Docker

The application can be run locally using Docker, without requiring installation of Go, Redis, or SQLite on the system.

> **Note:** Docker (or Docker Desktop) must be installed and running.

### Steps

```bash
git clone https://github.com/meghavx/go-miniurl.git

cd go-miniurl

docker-compose up --build

```
#### Once the containers are running, the application will be available at:<br>
http://localhost:8080

#### To stop the services:
```
docker-compose down
```
