# Demo Seed (Local Only)

File seed:
- `backend/seeds/001_demo_tenant.sql`

Kredensial demo:
- Email: `owner@pekan.local`
- Password: `password`
- Tenant ID: `11111111-1111-1111-1111-111111111111`

## Jalankan seed
Setelah semua migration `0001` s/d `0011` selesai, jalankan SQL seed ini ke database yang sama.

Helper script:
- Linux/macOS: `backend/scripts/apply_demo_seed.sh`
- Windows: `backend/scripts/apply_demo_seed.ps1`

## Quick API check
1. Login:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@pekan.local","password":"password","tenant_id":"11111111-1111-1111-1111-111111111111"}'
```

2. Get context:
```bash
curl http://localhost:8080/api/v1/me/context \
  -H "Authorization: Bearer <access_token>"
```

3. List accounts:
```bash
curl http://localhost:8080/api/v1/finance/accounts \
  -H "Authorization: Bearer <access_token>"
```
