package handler

import (
	"net/http"

	"snipvault/serverless"
)

// Handler is the Vercel Go Function entrypoint.
func Handler(w http.ResponseWriter, r *http.Request) {
	serverless.Handler(w, r)
}
