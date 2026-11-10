# SiteTrack API — Bruno collection

Import this folder into [Bruno](https://www.usebruno.com) (Open Collection).

- Environment: `Local` (edit `environments/Local.bru`).
- Auth is cookie based: run **01-auth / Login** first. Bruno keeps the
  `access_token` / `refresh_token` cookies for the rest of the session.
- IDs are captured automatically by the Create requests (`projectId`,
  `taskId`, `materialId`, `invitationId`, `deliveryId`, `roleId`), so run the
  collection top to bottom.
- Seed data: `seed.user1@example.com` / `Password123!`. Register a
  `company_store_manager` and a `super_admin` with **Register** to test the
  company-store and admin endpoints.
