package http

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// maxJSONBodyBytes caps the request body of every JSON endpoint — generous for a note's content
// while still bounding how much a client can force the server to buffer.
const maxJSONBodyBytes = 4 * 1024 * 1024

// writeJSON writes v as a JSON response body with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a {"error": message} JSON response with the given status code.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeJSON decodes a JSON request body capped at maxJSONBodyBytes.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(dst)
}

// pathUint64ID reads name out of the request path (e.g. {id} in /api/v1/notes/{id}) as a uint64,
// writing a 400 and returning ok=false if it's missing or not a valid non-negative integer.
func pathUint64ID(w http.ResponseWriter, r *http.Request, name string) (id uint64, ok bool) {
	id, err := strconv.ParseUint(r.PathValue(name), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return 0, false
	}

	return id, true
}
