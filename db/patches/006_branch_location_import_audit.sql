BEGIN;

CREATE TABLE IF NOT EXISTS public.ia_branch_location_mapping_import_audit (
    import_id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES public.ia_users(id),
    customer TEXT NOT NULL,
    mode TEXT NOT NULL,
    source_reference TEXT NOT NULL,
    source_sha256 TEXT NOT NULL,
    row_count INTEGER NOT NULL,
    previous_current_count INTEGER NOT NULL,
    closed_count INTEGER NOT NULL,
    inserted_count INTEGER NOT NULL,
    previous_mappings JSONB NOT NULL DEFAULT '[]'::JSONB,
    imported_mappings JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ia_branch_location_mapping_import_audit_mode_check
        CHECK (mode = 'replace'),
    CONSTRAINT ia_branch_location_mapping_import_audit_source_check
        CHECK (BTRIM(source_reference) <> '' AND BTRIM(source_sha256) <> ''),
    CONSTRAINT ia_branch_location_mapping_import_audit_counts_check
        CHECK (row_count > 0
            AND previous_current_count >= 0
            AND closed_count >= 0
            AND inserted_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_ia_branch_location_mapping_import_audit_owner_created
    ON public.ia_branch_location_mapping_import_audit (owner_user_id, created_at DESC);

COMMENT ON TABLE public.ia_branch_location_mapping_import_audit IS
    '分店地理 CSV replace 匯入稽核；保存來源雜湊、批次結果與匯入前 mapping snapshot，不碰 sales fact、branch dimension 或 IA Signals。';
COMMENT ON COLUMN public.ia_branch_location_mapping_import_audit.previous_mappings IS
    '匯入前該 tenant 的完整 mapping snapshot，供同日重跑或替換後追查。';
COMMENT ON COLUMN public.ia_branch_location_mapping_import_audit.imported_mappings IS
    '本批 CSV row snapshot；匯入器只保存 location 為 address，其他地理欄位不由 branch name 或地址推導。';

COMMIT;
