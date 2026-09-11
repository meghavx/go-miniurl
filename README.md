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
- **CI**: Jenkins

## Architecture Overview (HLD)
```
Client
  |
  | HTTP Request
  v
API Layer (Go)
  ├── Rate Limiter (Redis)
  ├── URL Resolver
  |     ├── Cache (Redis)
  |     |     └── ShortURL → LongURL
  |     |
  |     └── Bloom Filter
  |           └── Fast existence check for short codes
  |
  ├── Persistent Store (SQLite)
  |     └── URL mappings (source of truth)
  |
  └── Analytics Publisher (Async)
        └── Click events
```

## Run with Docker

The application can be run locally using Docker.

##### Note: Docker (or Docker Desktop) must be installed and running.

### Steps

```bash
git clone https://github.com/meghavx/go-miniurl.git

cd go-miniurl

docker-compose up --build

```
Once the containers are running, the application will be available at:

http://localhost:8080

#### To stop the services:
```
docker-compose down
```

## Jenkins CI

Jenkins is used to automate the project's CI workflow.

The pipeline:

1. Builds the test Docker image
2. Runs the project tests
3. Publishes the test results
4. Builds the application Docker image
5. Pushes the application image to Docker Hub

Jenkins runs separately from the application and can be started using its own Compose file:

```bash
docker compose -f docker-compose.jenkins.yml up -d --build
```

Once Jenkins is running, it will be available at:

http://localhost:8082

##### Notes: 
> 1. Jenkins requires an initial setup and configuration, including Docker Hub credentials (Personal Access Token) for pushing the application image.

> 2. Render deployment is kept separate from the Jenkins pipeline and is triggered manually.