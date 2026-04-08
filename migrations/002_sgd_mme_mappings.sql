-- VectorCore SMSC — Migration 002
-- S6c-to-SGd MME name mapping table.
-- Allows operators to translate the MME hostname returned by S6c (S6a FQDN)
-- to the correct SGd FQDN used for Diameter SGd delivery.

CREATE TABLE sgd_mme_mappings (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    s6c_result    TEXT NOT NULL UNIQUE,   -- MME hostname as returned by S6c (S6a FQDN)
    sgd_host      TEXT NOT NULL,          -- MME SGd FQDN to use for delivery
    enabled       BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER sgd_mme_mappings_notify
    AFTER INSERT OR UPDATE OR DELETE ON sgd_mme_mappings
    FOR EACH ROW EXECUTE FUNCTION notify_change();
