-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Grant privileges to pricingserver role
GRANT ALL PRIVILEGES ON SCHEMA public TO pricingserver;
GRANT CREATE ON SCHEMA public TO pricingserver;

-- Switch to pricingserver role and create tables
SET ROLE pricingserver;

-- Create schema and tables
CREATE TABLE IF NOT EXISTS contracts (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    parameters JSONB NOT NULL,
    created_at BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL,
    duration INTEGER NOT NULL
);

-- Reset role
RESET ROLE;

-- Set ownership
ALTER TABLE contracts OWNER TO pricingserver;

-- Set default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO pricingserver;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO pricingserver;
