package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task222-pdcluster/internal/model"
)

// writeJSON 输出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 把领域错误映射为 HTTP 状态码并输出。
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case model.IsNotFound(err):
		status = http.StatusNotFound
	case model.IsConflict(err):
		status = http.StatusConflict
	case model.IsInvalidState(err), errors.Is(err, model.ErrSealed):
		status = http.StatusConflict
	case errors.Is(err, model.ErrInvalidArgument):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON 解析请求体 JSON。
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// errorResponse 便捷构造错误响应体。
func errorResponse(err error) map[string]string { return map[string]string{"error": err.Error()} }
