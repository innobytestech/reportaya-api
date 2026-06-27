package errors

// ErrorDefinition describes a registered enterprise error.
type ErrorDefinition struct {
	Code        string           // Stable code like APP-GENRL01
	PublicMsg   string           // Safe for frontend
	Description string           // Internal detail for developers (logged only)
	HTTPStatus  int              // Recommended HTTP status
	Category    string           // validation, conflict, internal, etc.
	Sentinel    error            // Sentinel error for errors.Is matching
	TypeMatcher func(error) bool // Optional typed error matcher
}

// registry holds all enterprise error definitions.
// This map is populated at init() and is read-only after that.
//
// As domains are built, register their sentinel errors here mapping each to a
// stable code + HTTP status so handlers can call errors.HandleWithDefault.
var registry = make(map[string]ErrorDefinition)

// init registers the generic fallback error codes. Domain-specific codes are
// added by their own packages/files as modules are built.
func init() {
	register(ErrorDefinition{
		Code: "MS-GENRL01", PublicMsg: "Error interno del servidor",
		Description: "Unexpected internal server error with no specific domain mapping",
		HTTPStatus:  500, Category: CategoryInternal,
	})
	register(ErrorDefinition{
		Code: "MS-GENVA01", PublicMsg: "Datos de entrada inválidos",
		Description: "Struct tag validation failed on request payload",
		HTTPStatus:  400, Category: CategoryValidation,
	})
	register(ErrorDefinition{
		Code: "MS-GENVA02", PublicMsg: "Formato de la solicitud no es válido",
		Description: "Request body could not be parsed (malformed JSON or content-type mismatch)",
		HTTPStatus:  400, Category: CategoryValidation,
	})
	register(ErrorDefinition{
		Code: "MS-GENAU01", PublicMsg: "No autorizado",
		Description: "Generic unauthenticated fallback when no specific auth sentinel applies",
		HTTPStatus:  401, Category: CategoryAuth,
	})
	register(ErrorDefinition{
		Code: "MS-GENAU02", PublicMsg: "No tienes permiso para realizar esta acción",
		Description: "Generic authorization/permission denied fallback",
		HTTPStatus:  403, Category: CategoryAuth,
	})
	register(ErrorDefinition{
		Code: "MS-GENGT01", PublicMsg: "Recurso no encontrado",
		Description: "Generic 404 fallback (route or resource not found)",
		HTTPStatus:  404, Category: CategoryNotFound,
	})
	register(ErrorDefinition{
		Code: "MS-GENCN01", PublicMsg: "Conflicto con el estado actual del recurso",
		Description: "Generic 409 conflict fallback",
		HTTPStatus:  409, Category: CategoryConflict,
	})
	register(ErrorDefinition{
		Code: "MS-GENUN01", PublicMsg: "Servicio no disponible temporalmente",
		Description: "Generic 503 unavailable fallback for downstream dependencies",
		HTTPStatus:  503, Category: CategoryUnavailable,
	})
}

// register adds a single error definition to the registry.
func register(def ErrorDefinition) {
	if def.Code == "" {
		panic("error code cannot be empty")
	}
	if _, exists := registry[def.Code]; exists {
		panic("duplicate error code: " + def.Code)
	}
	registry[def.Code] = def
}

// Get returns a registered error definition by code.
func Get(code string) (ErrorDefinition, bool) {
	def, ok := registry[code]
	return def, ok
}

// AllCodes returns a slice of all registered error codes.
func AllCodes() []string {
	codes := make([]string, 0, len(registry))
	for code := range registry {
		codes = append(codes, code)
	}
	return codes
}

// Count returns the number of registered error codes.
func Count() int {
	return len(registry)
}
