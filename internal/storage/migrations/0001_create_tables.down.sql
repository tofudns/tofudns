-- Drop DNS records table
DROP TABLE IF EXISTS coredns_records;
DROP SEQUENCE IF EXISTS coredns_records_id_seq;

-- Drop OTP table
DROP TABLE IF EXISTS otp_codes;

-- Drop users table
DROP TABLE IF EXISTS users;

-- Drop UUID extension
DROP EXTENSION IF EXISTS "uuid-ossp";
