DROP INDEX IF EXISTS idx_reimbursements_receipt_public_id;

ALTER TABLE reimbursements
DROP CONSTRAINT reimbursements_receipt_metadata_complete,
DROP COLUMN receipt_delivery_type,
DROP COLUMN receipt_resource_type,
DROP COLUMN receipt_format,
DROP COLUMN receipt_asset_id,
DROP COLUMN receipt_public_id;
