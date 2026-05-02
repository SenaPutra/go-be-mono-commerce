CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO admin_users (name, email, password_hash, role, is_active)
VALUES (
    'Default Super Admin',
    'admin@example.com',
    crypt('admin12345', gen_salt('bf')),
    'SUPER_ADMIN',
    TRUE
)
ON CONFLICT (email) DO NOTHING;
