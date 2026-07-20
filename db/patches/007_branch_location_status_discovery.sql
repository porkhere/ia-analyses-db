BEGIN;

ALTER TABLE public.ia_branch_location_mapping
ADD COLUMN IF NOT EXISTS location_status TEXT NOT NULL DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS source_url TEXT,
ADD COLUMN IF NOT EXISTS fetched_at TIMESTAMPTZ,
ADD CONSTRAINT ia_branch_location_mapping_location_status_check
    CHECK (location_status IN ('pending', 'matched', 'not_found', 'needs_review'));

COMMENT ON COLUMN public.ia_branch_location_mapping.location_status IS
    '分店地址發現狀態: pending=尚未查詢, matched=確定命中, not_found=未找到, needs_review=需人工複核。';
COMMENT ON COLUMN public.ia_branch_location_mapping.source_url IS
    '地址查詢的來源 URL 或標準資源識別符，用於稽核與追溯。';
COMMENT ON COLUMN public.ia_branch_location_mapping.fetched_at IS
    '地址查詢實際執行的時刻，供重試與時間序列追蹤。';

COMMIT;
