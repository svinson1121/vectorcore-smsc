ALTER TABLE smpp_clients ADD COLUMN IF NOT EXISTS source_addr_ton INT;
ALTER TABLE smpp_clients ADD COLUMN IF NOT EXISTS source_addr_npi INT;
ALTER TABLE smpp_clients ADD COLUMN IF NOT EXISTS dest_addr_ton INT;
ALTER TABLE smpp_clients ADD COLUMN IF NOT EXISTS dest_addr_npi INT;
