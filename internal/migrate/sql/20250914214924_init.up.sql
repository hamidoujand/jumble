CREATE TABLE users(
    id  UUID PRIMARY KEY NOT NULL,
    name VARCHAR(200) NOT NULL, 
    email VARCHAR(300) NOT NULL UNIQUE,
    roles TEXT[] NOT NULL, 
    password_hash TEXT NOT NULL, 
    enabled BOOLEAN NOT NULL, 
    department VARCHAR(100) NULL, 
    created_at TIMESTAMP WITH TIME ZONE NOT NULL, 
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
)