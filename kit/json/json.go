package json

import (
	"net/http"

	"github.com/bytedance/sonic"
)

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := sonic.ConfigDefault.NewEncoder(w).Encode(data); err != nil {
		return
	}
}

func DecodeJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	return sonic.ConfigDefault.NewDecoder(r.Body).Decode(dest)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
