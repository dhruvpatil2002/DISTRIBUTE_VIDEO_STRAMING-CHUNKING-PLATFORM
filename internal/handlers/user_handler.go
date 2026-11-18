
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/auth"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type registerRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	AdminCode string `json:"admin_code,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

func writeUserJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("JSON response error: %v", err)
	}
}

func decodeUserJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst interface{},
) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		log.Printf(
			"JSON ERROR: method=%s path=%s error=%v",
			r.Method,
			r.URL.Path,
			err,
		)
		return err
	}

	if err := decoder.Decode(new(interface{})); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}

		log.Printf(
			"JSON TRAILING ERROR: method=%s path=%s error=%v",
			r.Method,
			r.URL.Path,
			err,
		)
		return err
	}

	return nil
}

func (h *UserHandler) Register(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req registerRequest

	if err := decodeUserJSON(w, r, &req); err != nil {
		writeUserJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}
      log.Printf(
        "REGISTER REQUEST: email=%s role=%s",
        req.Email,
        req.Role,
    )


	user, err := h.userService.Register(
		r.Context(),
		service.RegisterInput{
			Name:      req.Name,
			Email:     req.Email,
			Password:  req.Password,
			Role:      req.Role,
			AdminCode: req.AdminCode,
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			writeUserJSON(w, http.StatusConflict, map[string]string{
				"error": "email already registered",
			})

		case errors.Is(err, service.ErrInvalidInput):
			writeUserJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid registration details or role",
			})

		case errors.Is(err, service.ErrAdminRegistrationDenied):
			writeUserJSON(w, http.StatusForbidden, map[string]string{
				"error": "admin registration not authorized",
			})

		default:
			log.Printf("REGISTER ERROR: %v", err)

			writeUserJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "registration failed",
			})
		}
		return
	}

	writeUserJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "user registered successfully",
		"user":    user,
	})
}

func (h *UserHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req loginRequest

	if err := decodeUserJSON(w, r, &req); err != nil {

        log.Printf(
        "LOGIN DECODE ERROR: method=%s path=%s error=%v",
        r.Method,
        r.URL.Path,
        err,
    )

		writeUserJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.userService.Login(
		r.Context(),
		service.LoginInput{
			Email:    req.Email,
			Password: req.Password,
		},
	)

	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeUserJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid email or password",
			})
			return
		}

		log.Printf("LOGIN ERROR: %v", err)

		writeUserJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "login failed",
		})
		return
	}

	writeUserJSON(w, http.StatusOK, result)
}

func (h *UserHandler) Me(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := auth.GetUserID(r.Context())

	if !ok {
		writeUserJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "authentication required",
		})
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), userID)

	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeUserJSON(w, http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
			return
		}

		log.Printf("GET USER ERROR: %v", err)

		writeUserJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch user",
		})
		return
	}

	writeUserJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

func (h *UserHandler) GetAllUsers(
	w http.ResponseWriter,
	r *http.Request,
) {
	page := 1
	limit := 20

	if p := r.URL.Query().Get("page"); p != "" {
		value, err := strconv.Atoi(p)

		if err != nil || value < 1 || value > 1000000 {
			writeUserJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid page number",
			})
			return
		}

		page = value
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		value, err := strconv.Atoi(l)

		if err != nil || value < 1 || value > 100 {
			writeUserJSON(w, http.StatusBadRequest, map[string]string{
				"error": "limit must be between 1 and 100",
			})
			return
		}

		limit = value
	}

	users, total, err := h.userService.GetAllUsers(
		r.Context(),
		page,
		limit,
	)

	if err != nil {
		log.Printf("GET ALL USERS ERROR: %v", err)

		writeUserJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch users",
		})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	writeUserJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"current_page": page,
			"per_page":     limit,
			"total_users":  total,
			"total_pages":  totalPages,
		},
	})
}

func (h *UserHandler) UpdateRole(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := r.PathValue("id")

	if userID == "" {
		writeUserJSON(w, http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
		return
	}

	var req updateRoleRequest

	if err := decodeUserJSON(w, r, &req); err != nil {
		writeUserJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.userService.UpdateUserRole(
		r.Context(),
		userID,
		req.Role,
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeUserJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid role",
			})

		case errors.Is(err, service.ErrUserNotFound):
			writeUserJSON(w, http.StatusNotFound, map[string]string{
				"error": "user not found",
			})

		default:
			log.Printf("UPDATE ROLE ERROR: %v", err)

			writeUserJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to update role",
			})
		}
		return
	}

	writeUserJSON(w, http.StatusOK, map[string]interface{}{
		"message": "role updated successfully",
		"user":    user,
	})
}
