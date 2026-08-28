-- Integrity the contractor register was missing.
--
-- gov_contractors.code reads like an identifier and is used like one, but
-- nothing enforced it: CTR-0001 was issued to two different people and the
-- duplicate came across from the monolith unchallenged. Within an hour of this
-- index existing it caught a loader about to recreate that duplicate.
--
-- The index is PARTIAL because three contractors legitimately carry a blank
-- code. A full UNIQUE would conflate "no code yet" with "duplicate code" and
-- reject them.
CREATE UNIQUE INDEX IF NOT EXISTS gov_contractors_code_unique
    ON public.gov_contractors (code)
 WHERE btrim(coalesce(code, '')) <> '';

-- gov_contracts.contractor_id and gov_valuations.contractor_id had no foreign
-- key, so a contractor could be deleted out from under a live contract with no
-- error. RESTRICT rather than CASCADE: a contract must not lose its
-- counterparty, and the deletion is what should fail.
ALTER TABLE public.gov_contracts
  DROP CONSTRAINT IF EXISTS gov_contracts_contractor_id_fkey;
ALTER TABLE public.gov_contracts
  ADD CONSTRAINT gov_contracts_contractor_id_fkey FOREIGN KEY (contractor_id)
  REFERENCES public.gov_contractors (id) ON DELETE RESTRICT;

ALTER TABLE public.gov_valuations
  DROP CONSTRAINT IF EXISTS gov_valuations_contractor_id_fkey;
ALTER TABLE public.gov_valuations
  ADD CONSTRAINT gov_valuations_contractor_id_fkey FOREIGN KEY (contractor_id)
  REFERENCES public.gov_contractors (id) ON DELETE RESTRICT;
