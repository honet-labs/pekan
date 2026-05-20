# Postman Usage

Files:
- `docs/postman/PEKAN-API.postman_collection.json`
- `docs/postman/PEKAN-LOCAL.postman_environment.json`

## Steps
1. Import collection and environment.
2. Select environment `PEKAN Local`.
3. Run `Auth > Login`.
4. Copy `access_token` and `refresh_token` from response into environment variables.
5. Run `Auth > Logout All` if you want to invalidate all sessions for the current user.
6. Run other requests (finance master, transactions, entitlement).
7. For attachment scan status update, use `Finance Transactions > Set Attachment Scan Status`.

## Notes
- Collection uses `{{base_url}}`, `{{tenant_id}}`, `{{access_token}}`, `{{refresh_token}}`, `{{transaction_id}}`, `{{attachment_id}}`.
- Default base URL points to `http://localhost:8080/api/v1`.
