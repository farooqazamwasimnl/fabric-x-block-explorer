/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"net/http"
)

// Router returns the HTTP handler for the API.
func (a *API) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /blocks/height", a.GetBlockHeight)
	mux.HandleFunc("GET /blocks/{block_num}", a.GetBlockByNumber)
	mux.HandleFunc("GET /tx/{tx_id_hex}", a.GetTxByID)
	mux.HandleFunc("GET /healthz", a.HealthHandler)

	swaggerFS := http.FileServer(http.Dir("./pkg/swagger/ui"))
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", swaggerFS))

	mux.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./pkg/swagger/swagger.yaml")
	})

	return mux
}
