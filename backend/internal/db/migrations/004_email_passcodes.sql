-- +goose Up
CREATE TABLE email_passcodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_email_passcodes_email_created_at ON email_passcodes(email, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS email_passcodes;
