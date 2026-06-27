# Categories API

## Overview

The Categories API provides read-only access to the catalog of infrastructure issue categories.
This catalog serves as the foundation for report creation (Feature A2) and map filtering (Feature A4).

---

## Endpoint: List Categories

**Method:** `GET`

**Path:** `/api/categories`

**Authentication:** Public (no API key or JWT required in MVP)

**Description:** Returns the list of all active infrastructure report categories, ordered by name in ascending order.

### Request

No request body or parameters required.

**Example Request:**
```bash
curl -X GET "http://localhost:3000/api/categories"
```

### Response

**Status Code:** `200 OK`

**Content-Type:** `application/json`

The response uses the standard `APIResponse` envelope defined in `pkg/response`:

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

### Response Fields

Each category object in the `results` array contains:

- **`slug`** (string): A URL-safe, stable identifier for the category. Unique within the catalog. Used as the canonical key for report creation and map filtering. Example: `"baches"`.
- **`name`** (string): Human-readable label of the category. Example: `"Baches"`.
- **`icon`** (string): Icon identifier for UI rendering. Example: `"alert-circle"`.
- **`color`** (string): Hex color code for visual representation. Format: `"#RRGGBB"`. Example: `"#FF6347"`.

### Notes

- **Stable Contract:** The response structure (field names and types) is considered stable and will not change without a major version bump. The `slug` field is the recommended key for frontend caching and filtering logic.
- **Active Categories Only:** Only categories with `is_active = true` are returned. Inactive or logically deleted categories are excluded.
- **Deterministic Order:** Results are always ordered by `name` (ascending) to ensure consistent pagination and client-side caching.
- **No Pagination:** The MVP returns all active categories in a single response (expected ~12 items). Pagination may be added in future iterations if the catalog grows.
- **Error Handling:** On server error, the response will have `success: false` and an error message in the `error` field.

---

## Example Use Cases

### Creating a Report (Feature A2)

A report creation form uses the category list to build a dropdown:

```javascript
// Frontend example (pseudo-code)
const categories = await fetch('/api/categories').then(r => r.json()).then(r => r.results);
const dropdown = categories.map(cat => ({
  value: cat.slug,        // stable key for form submission
  label: cat.name,        // displayed to user
  icon: cat.icon,
  color: cat.color
}));
```

### Map Filtering (Feature A4)

The map module uses slugs to filter markers by category:

```javascript
// Frontend example
const activeCategories = ['baches', 'luminarias']; // user selection
const markers = allMarkers.filter(m => activeCategories.includes(m.categorySlug));
```

---

## Implementation Notes

- **Database:** Categories are stored in PostgreSQL table `categories` with a UNIQUE constraint on `slug`.
- **Soft-Delete:** The table uses soft-delete via `deleted_at` column. Only rows with `is_active = true` and `deleted_at IS NULL` are returned.
- **Seeding:** All 12 categories are seeded via migration `000002_categories.up.sql` on application startup.
- **ORM:** Data access is implemented via GORM with the repository pattern.
- **Concurrency:** The endpoint is safe for concurrent requests; read operations do not acquire locks.

---

## Future Extensions (Out of Scope for MVP)

- **Per-city Scoping:** Categories filtered by city or administrative zone (planned for A7).
- **Localization:** Multi-language category names and descriptions (planned for L1).
- **Admin Management:** CRUD endpoints for category administration (planned for B2).
- **Pagination:** If the catalog exceeds ~50 items, pagination may be introduced.
- **Caching Headers:** HTTP caching headers (ETag, Cache-Control) may be added after deployment.
