package docs

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec []byte

// Register mendaftarkan rute dokumentasi API ke router ServeMux utama.
// Rute yang didaftarkan:
//   - GET /openapi.yaml : Menyajikan file mentah spesifikasi OpenAPI 3.0
//   - GET /docs         : Antarmuka dokumentasi modern & interaktif via Scalar
//   - GET /swagger      : Antarmuka dokumentasi klasik via Swagger UI
func Register(mux *http.ServeMux) {
	// 1. Sajikan file mentah openapi.yaml
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(openAPISpec)
	})

	// 2. Sajikan Dokumentasi Interaktif Modern: SCALAR (Mendukung Dark Mode & Code Gen)
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, scalarHTML)
	})

	// 3. Sajikan Dokumentasi Standar: SWAGGER UI
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, swaggerHTML)
	})
}

const scalarHTML = `<!doctype html>
<html>
  <head>
    <title>ERP Retail Modular — API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" type="image/svg+xml" href="https://scalar.com/favicon.svg" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script id="api-reference" data-url="/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>ERP Retail Modular — Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5/favicon-32x32.png" />
  <style>
    body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/openapi.yaml',
        dom_id: '#swagger-ui',
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout",
        deepLinking: true
      });
    };
  </script>
</body>
</html>`
