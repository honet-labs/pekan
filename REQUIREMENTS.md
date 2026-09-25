# System Requirements & Tech Stack - PEKAN

This document outlines the recommended specifications and technology stack for deploying and running the PEKAN platform.

## 🚀 Technology Stack

### Backend
- **Language**: Go (Golang) 1.23+
- **Framework**: Standard Library + `chi` router (lightweight & fast)
- **Architecture**: Clean Architecture (Domain-Driven Design influenced)
- **Auth**: JWT (Stateless) with Refresh Token Rotation

### Frontend
- **Framework**: React 18+ with TypeScript
- **Build Tool**: Vite
- **State Management**: `useSyncExternalStore` (React built-in, lightweight & reactive)
- **Styling**: Vanilla CSS (Premium Custom Design)

### Infrastructure & Storage
- **Database**: PostgreSQL 16+ (Utilizes Schema-per-Tenant isolation)
- **Caching/Rate Limiting**: Redis 7+
- **Reverse Proxy**: Nginx
- **Service Management**: Systemd (Linux)
- **Containerization**: Docker & Docker Compose (Optional but recommended for DB/Redis)

---

## 💻 Hardware Specifications

| Specification | Minimum (Dev/Starter) | Recommended (SaaS Production) | Enterprise (High Load) |
| :--- | :--- | :--- | :--- |
| **vCPU** | 1 Core | 2 Cores | 4+ Cores |
| **RAM** | 2 GB | 4 GB | 8 GB+ |
| **Storage** | 20 GB SSD | 50 GB NVMe | 100 GB+ (Scalable) |
| **OS** | Ubuntu 22.04+ / Debian 12 | Ubuntu 24.04 LTS | RHEL / Rocky Linux 9 |
| **Network** | 100 Mbps | 1 Gbps | 1 Gbps+ |

---

## 🏗️ Deployment Recommendations

### 1. Virtual Machine (VPS/Cloud Instance) - **Recommended**
The current `install_server.sh` is optimized for a standalone VM. This is the simplest and most cost-effective way to run the SaaS.
- **Pros**: Easy to manage, low overhead, automatic systemd service handling.
- **Cons**: Manual scaling (Vertical).

### 2. Docker Compose
Use Docker Compose to manage the database and Redis while running the Go binary natively or in a container.
- **Pros**: Consistency across environments, easy updates.
- **Cons**: Slightly more resource overhead than native.

### 3. Kubernetes (K8s)
Suitable if you have thousands of tenants and need to scale the backend horizontally.
- **Pros**: High availability, horizontal scaling, auto-healing.
- **Cons**: High complexity, requires managed K8s service (EKS/GKE).

---

## 🛠️ Software Dependencies
- **PostgreSQL**: Must support `search_path` and multiple schemas.
- **Redis**: Required for IP-based rate limiting and session tracking.
- **OpenSSL**: Used for JWT secret generation.
- **Node.js 20+**: Required only for building the frontend bundle.

---

## 🔒 Security Requirements
- **SSL/TLS**: Mandatory for production (Let's Encrypt recommended).
- **Firewall**: Ports 80 (HTTP), 443 (HTTPS), and 8080 (API) should be managed.
- **Fail2Ban**: Recommended for mitigating brute-force attacks on `/auth/login`.

---

## 🔄 Maintenance & Migrations

### Global Migrations
Standard SQL migrations in `backend/migrations/*.sql` apply to the `public` schema (global tables like `tenants`, `users`).

### Tenant Migrations
Tenant-specific logic and isolated data tables are updated using the migration utility:
```bash
# Run from backend directory
DATABASE_URL="your-db-url" go run ./scripts/migrate_tenants.go
```
This utility automatically iterates through all active tenant schemas and applies updates from `backend/migrations/tenant/*.sql`.
