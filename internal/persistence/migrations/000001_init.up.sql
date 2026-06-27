-- 000001_init — kernel del esqueleto.
--
-- Habilita las extensiones base que usa la arquitectura (UUIDs y búsqueda
-- por trigramas). Agrega aquí (o en migraciones siguientes) el esquema real
-- de reportaya: usuarios, roles/permisos RBAC y la tabla de outbox de auditoría.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
