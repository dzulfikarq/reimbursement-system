-- 000007 :: drop departments — manager approval scope becomes all claims
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_department_id_fkey;
DROP INDEX IF EXISTS idx_users_department;
ALTER TABLE users DROP COLUMN IF EXISTS department_id;
DROP TABLE IF EXISTS departments;
