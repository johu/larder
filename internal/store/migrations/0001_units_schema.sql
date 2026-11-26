-- Units are reference data: language-neutral rows in unit, per-language
-- wording in unit_text. Base units are g (mass) and ml (volume).
CREATE TABLE unit (
  code TEXT NOT NULL PRIMARY KEY,
  dimension TEXT NOT NULL CHECK (
    dimension IN ('mass', 'volume', 'count', 'approx')
  ),
  base_factor REAL,
  sort_order INTEGER NOT NULL,
  -- mass and volume units convert to the base unit; count and approx
  -- units have no factor.
  CHECK (
    (dimension IN ('mass', 'volume')) = (base_factor IS NOT NULL)
  ),
  CHECK (
    base_factor IS NULL
    OR base_factor > 0
  )
);

-- No CHECK on lang: the set of supported languages lives in the code.
CREATE TABLE unit_text (
  code TEXT NOT NULL REFERENCES unit (code),
  lang TEXT NOT NULL,
  name TEXT NOT NULL CHECK (name <> ''),
  name_plural TEXT NOT NULL CHECK (name_plural <> ''),
  abbreviation TEXT CHECK (
    abbreviation IS NULL
    OR abbreviation <> ''
  ),
  PRIMARY KEY (code, lang)
) WITHOUT ROWID;
