SELECT 'CREATE DATABASE product_management_test'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'product_management_test'
)\gexec
