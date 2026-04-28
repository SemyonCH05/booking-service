INSERT INTO users (id, email, role, created_at)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'admin@gmail.com', 'admin', NOW()),
  ('00000000-0000-0000-0000-000000000002', 'user@gmail.com',  'user',  NOW())
ON CONFLICT (id) DO NOTHING;