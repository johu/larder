-- Shared by ListUnits and GetUnit; a WHERE clause is appended.
-- Placeholders: 1 = requested language, 2 = fallback language.
--
-- The fallback is decided per unit, not per column: if the unit has a text
-- row in the requested language, all of its columns are used, even a NULL
-- abbreviation (pt tbsp has none on purpose). Only a unit without a row in
-- that language gets the whole fallback row. Do not COALESCE per column,
-- or the English abbreviation would leak into such units.
SELECT
  u.code,
  u.dimension,
  u.base_factor,
  CASE
    WHEN requested.code IS NOT NULL THEN requested.name
    ELSE fallback.name
  END AS name,
  CASE
    WHEN requested.code IS NOT NULL THEN requested.name_plural
    ELSE fallback.name_plural
  END AS name_plural,
  CASE
    WHEN requested.code IS NOT NULL THEN requested.abbreviation
    ELSE fallback.abbreviation
  END AS abbreviation
FROM
  unit AS u
  LEFT JOIN unit_text AS requested ON requested.code = u.code
  AND requested.lang = ?
  LEFT JOIN unit_text AS fallback ON fallback.code = u.code
  AND fallback.lang = ?
