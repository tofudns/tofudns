-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create OTP codes table
CREATE TABLE otp_codes (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    code VARCHAR(8) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add indexes for OTP
CREATE INDEX idx_otp_codes_email ON otp_codes(email);
CREATE INDEX idx_otp_codes_code ON otp_codes(code);

-- Create DNS records table
CREATE SEQUENCE coredns_records_id_seq
    INCREMENT 1
    MINVALUE 1
    MAXVALUE 9223372036854775807
    START 1
    CACHE 1;

CREATE TABLE coredns_records (
    id bigint DEFAULT nextval('coredns_records_id_seq'::regclass) NOT NULL,
    user_id UUID NOT NULL,
    zone VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    ttl INT DEFAULT NULL,
    content TEXT,
    record_type VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Add index for faster zone lookups by user
CREATE INDEX idx_coredns_records_user_zone ON coredns_records(user_id, zone);

-- Add test users
INSERT INTO users (id, email) VALUES
('a89ecaee-799e-47f4-a483-3f14475e365d', 'george@noodles.gr'),
('47f44e6b-f91e-4a41-aea1-4b04f8c73a74', 'zf@sirodoht.com'),
('19f82521-437c-4d11-9abe-50979d53baa4', 'user2@example.com');

-- Add test DNS records
INSERT INTO coredns_records (user_id, zone, name, ttl, content, record_type) VALUES
('a89ecaee-799e-47f4-a483-3f14475e365d', 'example.org.', '', 30, '{"ip": "1.1.1.1"}', 'A'),
('a89ecaee-799e-47f4-a483-3f14475e365d', 'example.org.', '', 60, '{"ip": "1.1.1.0"}', 'A'),
('a89ecaee-799e-47f4-a483-3f14475e365d', 'example.org.', 'test', 30, '{"text": "hello"}', 'TXT'),
('a89ecaee-799e-47f4-a483-3f14475e365d', 'example.org.', 'mail', 30, '{"host" : "mail.example.org.","priority" : 10}', 'MX'),
('47f44e6b-f91e-4a41-aea1-4b04f8c73a74', 'user1domain.com.', '', 30, '{"ip": "2.2.2.2"}', 'A'),
('19f82521-437c-4d11-9abe-50979d53baa4', 'user2domain.com.', '', 30, '{"ip": "3.3.3.3"}', 'A');
