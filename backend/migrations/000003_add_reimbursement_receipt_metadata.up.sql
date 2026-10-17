ALTER TABLE reimbursements
ADD COLUMN receipt_public_id TEXT,
ADD COLUMN receipt_asset_id TEXT,
ADD COLUMN receipt_format TEXT,
ADD COLUMN receipt_resource_type TEXT,
ADD COLUMN receipt_delivery_type TEXT,
ADD CONSTRAINT reimbursements_receipt_metadata_complete CHECK (
    (receipt_public_id IS NULL
        AND receipt_asset_id IS NULL
        AND receipt_format IS NULL
        AND receipt_resource_type IS NULL
        AND receipt_delivery_type IS NULL)
    OR
    (receipt_public_id IS NOT NULL
        AND receipt_asset_id IS NOT NULL
        AND receipt_format IS NOT NULL
        AND receipt_resource_type IS NOT NULL
        AND receipt_delivery_type IS NOT NULL)
);

CREATE UNIQUE INDEX idx_reimbursements_receipt_public_id
ON reimbursements(receipt_public_id)
WHERE receipt_public_id IS NOT NULL;
