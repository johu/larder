-- Initial units (metric and common kitchen units).
-- This file must not be edited once it is on main:
-- corrections go into a new migration.
INSERT INTO
  unit (code, dimension, base_factor, sort_order)
VALUES
  ('g', 'mass', 1, 10),
  ('kg', 'mass', 1000, 20),
  ('ml', 'volume', 1, 30),
  ('l', 'volume', 1000, 40),
  ('tsp', 'volume', 5, 50),
  ('tbsp', 'volume', 15, 60),
  ('pinch', 'approx', NULL, 70),
  ('piece', 'count', NULL, 80),
  ('clove', 'count', NULL, 90),
  ('slice', 'count', NULL, 100),
  ('can', 'count', NULL, 110),
  ('pack', 'count', NULL, 120);

INSERT INTO
  unit_text (code, lang, name, name_plural, abbreviation)
VALUES
  ('g', 'en', 'gram', 'grams', 'g'),
  ('g', 'de', 'Gramm', 'Gramm', 'g'),
  ('g', 'pt', 'grama', 'gramas', 'g'),
  ('kg', 'en', 'kilogram', 'kilograms', 'kg'),
  ('kg', 'de', 'Kilogramm', 'Kilogramm', 'kg'),
  ('kg', 'pt', 'quilograma', 'quilogramas', 'kg'),
  ('ml', 'en', 'milliliter', 'milliliters', 'ml'),
  ('ml', 'de', 'Milliliter', 'Milliliter', 'ml'),
  ('ml', 'pt', 'mililitro', 'mililitros', 'ml'),
  ('l', 'en', 'liter', 'liters', 'l'),
  ('l', 'de', 'Liter', 'Liter', 'l'),
  ('l', 'pt', 'litro', 'litros', 'l'),
  ('tsp', 'en', 'teaspoon', 'teaspoons', 'tsp'),
  ('tsp', 'de', 'Teelöffel', 'Teelöffel', 'TL'),
  (
    'tsp',
    'pt',
    'colher de chá',
    'colheres de chá',
    NULL
  ),
  ('tbsp', 'en', 'tablespoon', 'tablespoons', 'tbsp'),
  ('tbsp', 'de', 'Esslöffel', 'Esslöffel', 'EL'),
  (
    'tbsp',
    'pt',
    'colher de sopa',
    'colheres de sopa',
    NULL
  ),
  ('pinch', 'en', 'pinch', 'pinches', NULL),
  ('pinch', 'de', 'Prise', 'Prisen', NULL),
  ('pinch', 'pt', 'pitada', 'pitadas', NULL),
  ('piece', 'en', 'piece', 'pieces', 'pc'),
  ('piece', 'de', 'Stück', 'Stück', 'Stk.'),
  ('piece', 'pt', 'unidade', 'unidades', 'un.'),
  ('clove', 'en', 'clove', 'cloves', NULL),
  ('clove', 'de', 'Zehe', 'Zehen', NULL),
  ('clove', 'pt', 'dente', 'dentes', NULL),
  ('slice', 'en', 'slice', 'slices', NULL),
  ('slice', 'de', 'Scheibe', 'Scheiben', NULL),
  ('slice', 'pt', 'fatia', 'fatias', NULL),
  ('can', 'en', 'can', 'cans', NULL),
  ('can', 'de', 'Dose', 'Dosen', NULL),
  ('can', 'pt', 'lata', 'latas', NULL),
  ('pack', 'en', 'pack', 'packs', NULL),
  ('pack', 'de', 'Packung', 'Packungen', NULL),
  ('pack', 'pt', 'embalagem', 'embalagens', NULL);
