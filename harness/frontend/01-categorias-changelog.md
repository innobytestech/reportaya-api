# Frontend Changelog — Feature 1 (A1) · Catálogo de Categorías

**Backend Status:** `Implemented`  
**Date:** 2026-06-27  
**Frontend Counterpart (Expected):** Feature A2 (Report Creation) and Feature A4 (Map Filtering)

---

## Resumen Ejecutivo

El backend ha completado la implementación del **catálogo de categorías** de infraestructura urbana.
El endpoint `GET /api/categories` (público, sin autenticación en MVP) devuelve la lista de 12 categorías
predefinidas, cada una identificada por un `slug` único y estable que debe usarse como clave canónica
en el frontend para filtros, dropdowns y persistencia de estado de usuario.

---

## 📋 Lo nuevo que hay para implementar (Contrato Backend)

### Endpoint

**`GET /api/categories`** (Público, sin autenticación requerida)

**URL Base:** `http://localhost:3000/api/categories` (desarrollo)

**Response HTTP 200:**

```json
{
  "success": true,
  "results": [
    {
      "slug": "aguas-negras",
      "name": "Aguas Negras",
      "icon": "droplet",
      "color": "#8B4513"
    },
    {
      "slug": "animal-muerto",
      "name": "Animal Muerto",
      "icon": "skull",
      "color": "#696969"
    },
    {
      "slug": "arboles",
      "name": "Árboles",
      "icon": "tree",
      "color": "#228B22"
    },
    {
      "slug": "banquetas",
      "name": "Banquetas",
      "icon": "square",
      "color": "#A9A9A9"
    },
    {
      "slug": "basura",
      "name": "Basura",
      "icon": "trash",
      "color": "#FFD700"
    },
    {
      "slug": "baches",
      "name": "Baches",
      "icon": "alert-circle",
      "color": "#FF6347"
    },
    {
      "slug": "drenaje",
      "name": "Drenaje",
      "icon": "water",
      "color": "#4169E1"
    },
    {
      "slug": "fuga-agua",
      "name": "Fuga de Agua",
      "icon": "droplets",
      "color": "#1E90FF"
    },
    {
      "slug": "grafiti",
      "name": "Grafiti",
      "icon": "pen-tool",
      "color": "#800080"
    },
    {
      "slug": "luminarias",
      "name": "Luminarias",
      "icon": "lamp",
      "color": "#FFD700"
    },
    {
      "slug": "semaforo",
      "name": "Semáforo",
      "icon": "traffic-light",
      "color": "#FF0000"
    },
    {
      "slug": "senaletica",
      "name": "Señalética",
      "icon": "sign",
      "color": "#DC143C"
    }
  ],
  "error": null
}
```

### Estructura de Datos (Contract)

Cada categoría en `results` contiene **exactamente estos campos** (orden alfabético en el código):

| Campo | Tipo | Descripción | Notas |
|-------|------|-------------|-------|
| `slug` | `string` | Identificador URL-safe único | **CLAVE CANÓNICA** para filtros, dropdowns, y persistencia de estado. Ejemplo: `"baches"`. Nunca cambia. |
| `name` | `string` | Etiqueta legible en español | Usada en UI. Ejemplo: `"Baches"`. |
| `icon` | `string` | Nombre del icono (p.ej. Feather, Material Icons) | Para renderizar en la UI. Ejemplo: `"alert-circle"`. |
| `color` | `string` | Código hexadecimal (`#RRGGBB`) | Para colorear markers, chips, badges. Ejemplo: `"#FF6347"`. |

### Garantías del Backend

- **Siempre 12 categorías:** La respuesta contiene exactamente 12 elementos (seeded en migración).
- **Siempre activas:** Solo se devuelven categorías con `is_active = true`. Categorías desactivadas (soft-delete) no aparecen.
- **Siempre ordenadas:** Orden determinista por `name` ascendente (alfabético). Ejemplo: "Aguas Negras" antes que "Baches".
- **Siempre completo:** Todos los campos (`slug`, `name`, `icon`, `color`) están siempre presentes. Nunca `null` ni omitidos.
- **Estable:** El contrato (estructura y nombres de campos) no cambiará sin aviso. Los `slug` son IDs permanentes.

### Errores Posibles

| Status | Caso | Respuesta |
|--------|------|-----------|
| `200` | Éxito | `{"success": true, "results": [...], "error": null}` |
| `500` | Error del servidor | `{"success": false, "results": null, "error": "Internal Server Error"}` |

En MVP no hay validación de request (sin parámetros). Los errores 4xx (4xx) no aplican.

---

## 🛠️ Tareas que surgen de esta nueva implementación (Frontend Checklist)

### Capa de Modelos / API

- [ ] **T1.1** Definir tipo TypeScript `Category` que mapee a los campos `(slug, name, icon, color)`.
- [ ] **T1.2** Crear servicio API `CategoryService` (o `useCategories()` hook) que haga `GET /api/categories` con error handling (retry, fallback, logging).
- [ ] **T1.3** Implementar almacenamiento en caché (memoria, localStorage o Zustand/Redux) indexado por `slug` para evitar re-fetches. Invalidar caché solo si el usuario explícitamente refresca.
- [ ] **T1.4** Documentar en archivo local `src/types/categories.ts` el contrato TypeScript importado del backend. Incluir comentarios de cuáles campos se usan dónde.

### Capa de UI / Componentes

- [ ] **T2.1** Crear componente `<CategoryPicker>` (dropdown / select) que use la lista de categorías del caché. El value del select es el `slug`, no el nombre.
- [ ] **T2.2** Crear componente `<CategoryBadge>` (visual chip) que renderice el `name`, icono (`icon`), y color (`color`) de una categoría dado su `slug`.
- [ ] **T2.3** Crear componente `<CategoryIcon>` que mapee los nombres de iconos del backend (p.ej. `"alert-circle"`) a componentes de ícono de tu librería (p.ej. Feather Icons, Material Icons).
- [ ] **T2.4** En formularios de creación de reportes (A2), usar `<CategoryPicker>` en el campo "Categoría". Validar que `slug` no esté vacío.
- [ ] **T2.5** En módulo de filtros del mapa (A4), crear controles multi-select (`<CategoryFilter>`) que permitan seleccionar múltiples slugs. Persistir selección en URL query params o state de aplicación.

### Integración con Otros Módulos

- [ ] **T3.1** En formulario de creación de reportes (A2): cuando el usuario selecciona una categoría, guardar el `slug` en el payload del POST (no el `name` ni el `id` del backend).
- [ ] **T3.2** En módulo de reportes existentes: cuando se muestre un reporte listado, renderizar su categoría usando `<CategoryBadge>` con el `slug` almacenado.
- [ ] **T3.3** En mapa (A4): cuando filtro es aplicado por usuario, mostrar solo markers cuya categoría (`slug`) está en la selección. Usar colores del backend para representar clusters por categoría.
- [ ] **T3.4** En detalles de reporte: mostrar el nombre y color de la categoría (enriquecida desde caché). Si por algún motivo el caché no tiene la categoría, hacer fallback fetch.

### Testing y Validación

- [ ] **T4.1** Escribir test unitario para `CategoryService`: mock de `GET /api/categories`, validar que parsea correctamente 12 elementos.
- [ ] **T4.2** Escribir test de componente para `<CategoryPicker>`: verificar que renderiza todos los slugs, que el valor seleccionado es un slug válido, que el select triggeriza callback correcto.
- [ ] **T4.3** Escribir test de integración: crear un reporte seleccionando una categoría en el picker, verificar que el payload contiene el `slug` correcto.
- [ ] **T4.4** Test manual (E2E): abrir la app, llamar `GET /api/categories`, verificar en DevTools que la respuesta contiene 12 elementos, que todos tienen `slug`/`name`/`icon`/`color`, que están alfabéticamente ordenados.

### Documentación

- [ ] **T5.1** Escribir README o guía en `docs/frontend/CATEGORIES.md` describiendo cómo usar la caché y los componentes.
- [ ] **T5.2** En archivo `src/services/CategoryService.ts` (o similar), incluir comentarios de qué es un `slug` y por qué se usa como clave (ej: "estable a través de actualizaciones, no es UUID").
- [ ] **T5.3** Documentar en Storybook o similar los componentes `<CategoryPicker>` y `<CategoryBadge>` con ejemplos de uso e historias.

---

## Notas Importantes para el Equipo Frontend

### El Slug es la Clave Canónica

**NO uses el `name` ni un índice interno como clave.** El `slug` es el identificador único y permanente:
- ✅ Correcto: `categorySlug: "baches"` en base de datos / formularios.
- ❌ Incorrecto: `categoryName: "Baches"` o `categoryIndex: 0`.

El backend **garantiza** que los slugs son únicos y estables (nunca cambian a menos que se retire una categoría).

### Caché y Refetch

Llama a `GET /api/categories` **una sola vez** al inicio de la app (o cuando el usuario abre un modal/form que lo necesite).
Almacena el resultado en caché (p.ej. Zustand, Redux, contexto React) indexado por `slug`.

Ejemplo:
```typescript
// caché indexado por slug
const categoryMap = new Map(
  categories.map(c => [c.slug, c])
);

// lookup rápido O(1)
const categoryColor = categoryMap.get("baches")?.color; // "#FF6347"
```

### Orden y Estabilidad

El backend **siempre** devuelve las categorías ordenadas alfabéticamente por `name`.
No ordenes nuevamente en el frontend; confía en el orden del servidor.
Esto asegura consistencia entre cliente y servidor, y facilita pruebas E2E reproducibles.

### Icono y Color

Los campos `icon` y `color` son sugerencias visuales del diseño:
- **`icon`:** Nombre de ícono compatible con tu librería (Feather, Material, etc.). Consulta el archivo `docs/api/categories.md` para la lista completa.
- **`color`:** Código hexadecimal seguro para CSS directo (`style={{ color: category.color }}`). Validado en el backend (formato `#RRGGBB`).

### Sin Autenticación (MVP)

En MVP, `GET /api/categories` es **pública**. No requiere token JWT ni API key.
Cuando se implemente seguridad (Feature B3), este endpoint seguirá siendo público (es un catálogo, no datos sensitivos).

---

## Referencias

- **Backend Docs:** `docs/api/categories.md`
- **Specs Completos:** `harness/specs/01-categorias/`
- **Implementación Backend:** `internal/domain/category/`
- **Migraciones:** `internal/persistence/migrations/000002_categories.{up,down}.sql`
