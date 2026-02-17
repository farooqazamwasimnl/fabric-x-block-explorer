package swagger

import (
	"net/http"
)

// Mount attaches Swagger UI + swagger.yaml to the given mux.
func Mount(mux *http.ServeMux) {
	// Serve Swagger UI static files
	swaggerFS := http.FileServer(http.Dir("./pkg/api/swagger-ui"))
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", swaggerFS))

	// Serve swagger.yaml
	mux.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./pkg/api/swagger.yaml")
	})

	// Optional: redirect /swagger → /swagger/
	mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
}
