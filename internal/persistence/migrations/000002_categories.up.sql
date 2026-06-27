-- 000002_categories — tabla de categorías de reportes de infraestructura urbana.
--
-- Crea la tabla categories con su índice único en slug (R6) y siembra las 12
-- categorías del catálogo aprobado (R1). gen_random_uuid() proviene de pgcrypto
-- habilitada en 000001_init.
CREATE TABLE IF NOT EXISTS categories (
    id         uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       varchar(50)  NOT NULL UNIQUE,
    name       varchar(100) NOT NULL,
    icon       varchar(50)  NOT NULL,
    color      varchar(7)   NOT NULL,
    is_active  boolean      NOT NULL DEFAULT true,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories (deleted_at);

INSERT INTO categories (slug, name, icon, color) VALUES
    ('baches',        'Baches',                         'pothole',       '#E11D48'),
    ('luminarias',    'Luminarias apagadas',            'lightbulb',     '#F59E0B'),
    ('fuga-agua',     'Fugas de agua',                  'water-drop',    '#0EA5E9'),
    ('basura',        'Acumulación de basura',          'trash',         '#65A30D'),
    ('drenaje',       'Drenaje / alcantarilla tapada',  'manhole',       '#7C3AED'),
    ('aguas-negras',  'Aguas negras / fuga de drenaje', 'sewage',        '#92400E'),
    ('semaforo',      'Semáforo descompuesto',          'traffic-light', '#DC2626'),
    ('senaletica',    'Señalización vial dañada',       'sign',          '#2563EB'),
    ('banquetas',     'Banquetas dañadas',              'sidewalk',      '#475569'),
    ('grafiti',       'Grafiti / vandalismo',           'spray-can',     '#DB2777'),
    ('arboles',       'Árboles caídos / poda',          'tree',          '#16A34A'),
    ('animal-muerto', 'Animal muerto en vía pública',   'paw',           '#57534E')
ON CONFLICT (slug) DO NOTHING;
