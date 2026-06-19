-- Monthly-report meta: the cross-cutting Challenges register and the Action-Item
-- tracker that accompany the Construction Department monthly report. These are
-- report-level narrative entities (not tied to a single contract), so they live
-- in their own period-scoped tables rather than on gov_contracts.
--
-- Follows the legacy-safe pattern: CREATE TABLE IF NOT EXISTS, then idempotent
-- seed INSERTs (deterministic ids + ON CONFLICT DO NOTHING) for the May-2026
-- report. Runs once per DB via the embedded migration runner on startup.

-- ----- Challenges register (issue -> affected -> recommended action -> owner) -----
CREATE TABLE IF NOT EXISTS gov_challenges (
    id          TEXT PRIMARY KEY,
    period      TEXT NOT NULL,
    seq         INT NOT NULL DEFAULT 0,
    category    TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    affected    TEXT NOT NULL DEFAULT '',
    priority    TEXT NOT NULL DEFAULT 'Medium',
    action      TEXT NOT NULL DEFAULT '',
    owner       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (period, seq)
);
CREATE INDEX IF NOT EXISTS idx_gov_challenges_period ON gov_challenges (period);

-- ----- Action-item tracker (priority -> responsible party -> target -> status) -----
CREATE TABLE IF NOT EXISTS gov_action_items (
    id         TEXT PRIMARY KEY,
    period     TEXT NOT NULL,
    seq        INT NOT NULL DEFAULT 0,
    priority   TEXT NOT NULL DEFAULT 'Medium',
    text       TEXT NOT NULL DEFAULT '',
    party      TEXT NOT NULL DEFAULT '',
    target     TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (period, seq)
);
CREATE INDEX IF NOT EXISTS idx_gov_action_items_period ON gov_action_items (period);


-- ----- Seed: May-2026 challenges -----
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-001', '2026-05', 1, 'Payment Delays', 'Multiple contractors report delayed IPC payments affecting remobilisation and material procurement', 'Matovu, DOASCORE, Sigi Patrick, PTAKA, Josira, Ochaya, Kazora', 'HIGH', 'Honour reviewed contracts; consultants to expedite review by delegating; Finance to set payment schedule', 'Finance / CEO') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-002', '2026-05', 2, 'Material Supply', 'Delayed delivery of hardcore, sand, blocks, culverts and specialised materials stalling progress', 'Matovu, Alex Matwa, Newell, PTAKA, Kazora', 'HIGH', 'Source alternative suppliers; set weekly delivery targets; stock critical materials on site', 'Procurement / Logistics') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-003', '2026-05', 3, 'Fleet Management & Availability', 'Availability and rampant breakdown of fleet; deployment without consulting the construction department', 'Newell, Doascore, Nowera, Sate, BAM, Matwa, Block Making Team', 'HIGH', 'Establish fleet booking system; dedicate equipment to active sites; set up in-house maintenance team', 'Equipment Manager / PM') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-004', '2026-05', 4, 'Workmanship & Quality', 'Unsatisfactory workmanship and poor sand delivered to site', 'All contractors', 'HIGH', 'Weekly quality meetings & trainings; hire 2 Civil Engineers (Nyabihoko + ACP); hire competent contractors; alternative sources', 'Procurement / PM') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-005', '2026-05', 5, 'Undefined Scope & Documentation', 'Most contracts lack defined scope, making administration difficult', 'All contractors', 'HIGH', 'CEO to intervene; proper contracts to be authored moving forward', 'CEO / GM / PM') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-006', '2026-05', 6, 'Design Indecision', 'Consultants and clients delaying finishing details, structural drawings and design approvals', 'Matovu (1,2,7), DOASCORE, Sigi Patrick', 'MEDIUM', 'Set hard deadlines for approvals; PM to escalate to CEO; consultants to provide interim guidance', 'Consultants / PM') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-007', '2026-05', 7, 'Weather Impact', 'Rainy season causing work stoppages on open-air civil and paving works', 'Matovu, DOASCORE', 'MEDIUM', 'Develop wet-weather schedule; cover sensitive works; pre-stock materials ahead of rains', 'Site Engineers / PM') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-008', '2026-05', 8, 'Financial / Value for Money', 'Absence of bidding leads to awarding contracts at premium prices and occasional overpayment', 'Inspire', 'MEDIUM', 'Use engineer''s estimates for benchmarking; adopt bidding protocols; set up award committee', 'CEO / PM') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-009', '2026-05', 9, 'Equipment Shortage', 'Lack of earthmoving equipment (backhoe, excavator) slowing excavation and trenching', 'James (Water), Peter Segawa, Newell', 'MEDIUM', 'Mobilise backhoe and excavator to priority sites; dedicate scheduled days; explore hire', 'Equipment Manager') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_challenges (id, period, seq, category, description, affected, priority, action, owner) VALUES ('GCHL-SEED-010', '2026-05', 10, 'Safety / Compliance', 'Cosmetics modification halted on PM safety instructions; UETCL pole conflict blocking security house', 'Maurice, Matovu (26)', 'MEDIUM', 'Issue written safety clearance; engage UETCL for pole relocation; document safety plan', 'Project Manager / Engineering') ON CONFLICT (period, seq) DO NOTHING;

-- ----- Seed: May-2026 action items -----
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-001', '2026-05', 1, 'CRITICAL', 'Release pending IPC payments to halted contractors (Matovu, DOASCORE, Kazora, PTAKA, Ochaya, Josira) to enable immediate remobilisation', 'Finance / Management', '10 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-002', '2026-05', 2, 'CRITICAL', 'Resolve and communicate final finishing design details for Main Gate & LHS perimeter wall (Matovu 1 & 2) and fountain control room (DOASCORE)', 'Consultants / PM', '08 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-003', '2026-05', 3, 'HIGH', 'Procure and deliver hardcore, solid blocks, silt-free sand and culverts to priority sites: Matovu (15,20,27), PTAKA, Kazora (4,5,6)', 'Procurement / Logistics', '12 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-004', '2026-05', 4, 'HIGH', 'Provide structural drawings for Branch-off gate & steel yard wall (Matovu 7) and resolve UETCL pole conflict (26)', 'Engineering / External', '15 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-005', '2026-05', 5, 'HIGH', 'Mobilise earthmoving equipment (backhoe & excavator) to James water project and Newell intake chamber', 'Equipment Manager', '10 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-006', '2026-05', 6, 'MEDIUM', 'Approve cosmetics modification safety plan and issue written clearance to Maurice for resumption', 'Project Manager', '08 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-007', '2026-05', 7, 'MEDIUM', 'Approve A-shaped units variation for NOWERA and issue outstanding payment for capsules / roofing works', 'PM / Finance', '15 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-008', '2026-05', 8, 'MEDIUM', 'Deliver undelivered lanterns to enable Matovu (21) to resume courtyard lantern installation', 'Procurement', '14 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-009', '2026-05', 9, 'MEDIUM', 'Provide facing bricks for Matovu RC shear wall (4) and machine for soil spreading at cement store (27)', 'Procurement / Site', '14 Jun 2026', 'PENDING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-010', '2026-05', 10, 'ROUTINE', 'Conduct weekly site progress reviews with all active contractors and update tracker status', 'PM / Site Engineers', 'Every Friday', 'ONGOING') ON CONFLICT (period, seq) DO NOTHING;
INSERT INTO gov_action_items (id, period, seq, priority, text, party, target, status) VALUES ('GACT-SEED-011', '2026-05', 11, 'ROUTINE', 'Monitor NUATU stone crusher retaining wall steel works and coordinate embedded plate delivery for July machine installation', 'Site Engineer', '30 Jun 2026', 'ONGOING') ON CONFLICT (period, seq) DO NOTHING;
