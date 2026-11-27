-- Placeholders: 3 and 4 = dimension (twice). An empty dimension means no
-- filter.
WHERE
  (
    ? = ''
    OR u.dimension = ?
  )
ORDER BY
  u.sort_order
