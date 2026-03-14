CREATE TABLE activity_codes (
    code        VARCHAR PRIMARY KEY,
    description VARCHAR NOT NULL
);

INSERT INTO activity_codes (code, description) VALUES
  ('01.1',  'Gajenje jednogodišnjih biljaka'),
  ('10.1',  'Prerada i konzerviranje mesa'),
  ('41.2',  'Izgradnja stambenih i nestambenih zgrada'),
  ('45.1',  'Trgovina motornim vozilima'),
  ('46.1',  'Posredovanje u trgovini'),
  ('47.1',  'Trgovina na malo u nespecijalizovanim prodavnicama'),
  ('56.1',  'Delatnost restorana i pokretnih ugostiteljskih objekata'),
  ('62.01', 'Računarsko programiranje'),
  ('62.02', 'Konsultantske delatnosti u oblasti informacione tehnologije'),
  ('64.19', 'Ostalo monetarno posredovanje'),
  ('69.1',  'Pravne delatnosti'),
  ('70.2',  'Konsultantske delatnosti u oblasti poslovanja'),
  ('85.3',  'Srednje obrazovanje'),
  ('86.1',  'Bolničke delatnosti'),
  ('96.0',  'Ostale lične uslužne delatnosti')
ON CONFLICT (code) DO NOTHING;
