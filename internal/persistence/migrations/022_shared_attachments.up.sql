-- A shared attachment store for documents whose owning service has no table of
-- its own for them.
--
-- ==================== WHY SHARED ====================
--
-- The 25 files belong to five different owners - purchase bills, projects,
-- payment certificates, work variations and general requests. Building five
-- per-service attachment tables for 3.4 MB would be five implementations of the
-- same thing. dms_attachments and pm_file_blobs already exist and are scoped to
-- their own services; this is the general case, and owner_service records which
-- service the row belongs to so it can be split out later without data loss.
--
-- ==================== BYTES LIVE IN OBJECT STORAGE ====================
--
-- storage_key is the object-store key, matching the convention dms_attachments
-- and pm_file_blobs already use.
--
-- It is NULLABLE on purpose. The object store exists but is not yet connected to
-- the services, so the bytes are still only in legacy_erp.attachment_blobs.
-- Writing a storage_key now would record a key that resolves to nothing, which
-- is worse than recording none: a null says "not uploaded yet" and can be found,
-- a dangling key says "uploaded" and cannot.
--
-- source_blob_id is where the bytes are until then. Once the store is connected,
-- upload each blob, stamp storage_key, and the column stops being the source of
-- truth without any row having lied in the meantime.
--
-- checksum is carried from the source so an upload can be verified rather than
-- assumed.


CREATE TABLE IF NOT EXISTS public.attachments (
    id             UUID PRIMARY KEY,
    owner_service  TEXT NOT NULL,
    owner_type     TEXT NOT NULL,
    -- deliberately text: owners are keyed differently across services (uuid in
    -- procurement, text in gov_*, a reference string in the source), so this is
    -- not a foreign key. owner_service plus owner_type say how to read it.
    owner_ref      TEXT NOT NULL,
    filename       TEXT NOT NULL,
    mime           TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes     BIGINT NOT NULL DEFAULT 0,
    checksum       TEXT,
    storage_key    TEXT,
    source_blob_id UUID,
    uploaded_by    TEXT NOT NULL DEFAULT '',
    uploaded_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- either it is in object storage, or we know where the bytes still are;
    -- a row that can point at neither is not an attachment.
    CONSTRAINT attachments_locatable CHECK (storage_key IS NOT NULL OR source_blob_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_attachments_owner   ON public.attachments (owner_service, owner_type, owner_ref);
CREATE INDEX IF NOT EXISTS idx_attachments_pending ON public.attachments (source_blob_id) WHERE storage_key IS NULL;

