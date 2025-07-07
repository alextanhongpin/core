package handler_test

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/handler"
	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
	"github.com/alextanhongpin/testdump/httpdump"
)

// Real-world example: User management API
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
	return cause.Map{
		"name":  cause.Required(r.Name).Err(),
		"email": cause.Required(r.Email).Err(),
	}.Err()
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r UpdateUserRequest) Validate() error {
	return cause.Map{
		"name":  cause.Required(r.Name).Err(),
		"email": cause.Required(r.Email).Err(),
	}.Err()
}

type GetUserResponse struct {
	User User `json:"user"`
}

type CreateUserResponse struct {
	User User `json:"user"`
}

type ListUsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// Mock service
type UserService struct {
	users  map[int]User
	nextID int
}

func NewUserService() *UserService {
	return &UserService{
		users:  make(map[int]User),
		nextID: 1,
	}
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	// Simulate duplicate email check
	for _, user := range s.users {
		if user.Email == req.Email {
			return User{}, cause.New(codes.Conflict, "user/email_exists", "Email already exists")
		}
	}

	user := User{
		ID:    s.nextID,
		Name:  req.Name,
		Email: req.Email,
	}
	s.users[s.nextID] = user
	s.nextID++

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id int) (User, error) {
	user, exists := s.users[id]
	if !exists {
		return User{}, cause.New(codes.NotFound, "user/not_found", "User not found").Wrap(sql.ErrNoRows)
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id int, req UpdateUserRequest) (User, error) {
	user, exists := s.users[id]
	if !exists {
		return User{}, cause.New(codes.NotFound, "user/not_found", "User not found")
	}

	// Check for email conflicts
	for userID, existingUser := range s.users {
		if userID != id && existingUser.Email == req.Email {
			return User{}, cause.New(codes.Conflict, "user/email_exists", "Email already exists")
		}
	}

	user.Name = req.Name
	user.Email = req.Email
	s.users[id] = user

	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id int) error {
	if _, exists := s.users[id]; !exists {
		return cause.New(codes.NotFound, "user/not_found", "User not found")
	}
	delete(s.users, id)
	return nil
}

func (s *UserService) List(ctx context.Context) ([]User, error) {
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	// Sort by ID for deterministic results in tests
	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})

	return users, nil
}

// Real-world controller implementation
type UserController struct {
	handler.BaseHandler
	userService *UserService
}

func NewUserController(userService *UserService) *UserController {
	return &UserController{
		BaseHandler: handler.BaseHandler{}.WithLogger(slog.Default()),
		userService: userService,
	}
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
	requestID := c.GetRequestID(r)
	c.SetRequestID(w, requestID)

	var req CreateUserRequest
	if err := c.ReadJSON(r, &req); err != nil {
		c.Next(w, r, err)
		return
	}

	user, err := c.userService.Create(r.Context(), req)
	if err != nil {
		c.Next(w, r, err)
		return
	}

	c.Body(w, CreateUserResponse{User: user}, http.StatusCreated)
}

func (c *UserController) GetUser(w http.ResponseWriter, r *http.Request) {
	requestID := c.GetRequestID(r)
	c.SetRequestID(w, requestID)

	// In a real application, you'd parse the ID from the URL path
	userID := 1
	if r.URL.Query().Get("id") == "999" {
		userID = 999 // Simulate not found
	}

	user, err := c.userService.GetByID(r.Context(), userID)
	if err != nil {
		c.Next(w, r, err)
		return
	}

	c.Body(w, GetUserResponse{User: user}, http.StatusOK)
}

func (c *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) {
	requestID := c.GetRequestID(r)
	c.SetRequestID(w, requestID)

	var req UpdateUserRequest
	if err := c.ReadJSON(r, &req); err != nil {
		c.Next(w, r, err)
		return
	}

	// In a real application, you'd parse the ID from the URL path
	userID := 1

	user, err := c.userService.Update(r.Context(), userID, req)
	if err != nil {
		c.Next(w, r, err)
		return
	}

	c.Body(w, CreateUserResponse{User: user}, http.StatusOK)
}

func (c *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	requestID := c.GetRequestID(r)
	c.SetRequestID(w, requestID)

	// In a real application, you'd parse the ID from the URL path
	userID := 1
	if r.URL.Query().Get("id") == "999" {
		userID = 999 // Simulate not found
	}

	if err := c.userService.Delete(r.Context(), userID); err != nil {
		c.Next(w, r, err)
		return
	}

	c.NoContent(w)
}

func (c *UserController) ListUsers(w http.ResponseWriter, r *http.Request) {
	requestID := c.GetRequestID(r)
	c.SetRequestID(w, requestID)

	users, err := c.userService.List(r.Context())
	if err != nil {
		c.Next(w, r, err)
		return
	}

	c.Body(w, ListUsersResponse{
		Users: users,
		Total: len(users),
	}, http.StatusOK)
}

// Integration tests
func TestUserController_Integration(t *testing.T) {
	userService := NewUserService()
	controller := NewUserController(userService)

	t.Run("create user success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"John Doe","email":"john@example.com"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Request-ID", "test-request-123")

		hd := httpdump.HandlerFunc(t, controller.CreateUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}

		if w.Header().Get("X-Request-ID") != "test-request-123" {
			t.Error("Request ID should be set in response header")
		}
	})

	t.Run("create user validation error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"","email":"john@example.com"}`))
		r.Header.Set("Content-Type", "application/json")

		hd := httpdump.HandlerFunc(t, controller.CreateUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("create user duplicate email", func(t *testing.T) {
		// First, create a user
		userService.Create(context.Background(), CreateUserRequest{
			Name:  "Jane Doe",
			Email: "jane@example.com",
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"John Doe","email":"jane@example.com"}`))
		r.Header.Set("Content-Type", "application/json")

		hd := httpdump.HandlerFunc(t, controller.CreateUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})

	t.Run("get user success", func(t *testing.T) {
		// Create a user first
		user, _ := userService.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		r.Header.Set("X-Correlation-ID", "correlation-456")

		hd := httpdump.HandlerFunc(t, controller.GetUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		if w.Header().Get("X-Request-ID") != "correlation-456" {
			t.Error("Correlation ID should be set as request ID in response header")
		}

		_ = user // Use the user variable to avoid unused variable error
	})

	t.Run("get user not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/999?id=999", nil)

		hd := httpdump.HandlerFunc(t, controller.GetUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("update user success", func(t *testing.T) {
		// Create a user first
		userService.Create(context.Background(), CreateUserRequest{
			Name:  "Update Test",
			Email: "update@example.com",
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/users/1", strings.NewReader(`{"name":"Updated Name","email":"updated@example.com"}`))
		r.Header.Set("Content-Type", "application/json")

		hd := httpdump.HandlerFunc(t, controller.UpdateUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("delete user success", func(t *testing.T) {
		// Create a user first
		userService.Create(context.Background(), CreateUserRequest{
			Name:  "Delete Test",
			Email: "delete@example.com",
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/1", nil)

		hd := httpdump.HandlerFunc(t, controller.DeleteUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("delete user not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/999?id=999", nil)

		hd := httpdump.HandlerFunc(t, controller.DeleteUser)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("list users", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users", nil)

		hd := httpdump.HandlerFunc(t, controller.ListUsers)
		hd.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// Benchmark tests
func BenchmarkBaseHandler_ReadJSON(b *testing.B) {
	base := handler.BaseHandler{}
	body := `{"name":"John Doe","email":"john@example.com"}`

	b.ResetTimer()
	for b.Loop() {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		var req CreateUserRequest
		base.ReadJSON(r, &req)
	}
}

func BenchmarkBaseHandler_JSON(b *testing.B) {
	base := handler.BaseHandler{}
	data := CreateUserResponse{
		User: User{
			ID:    1,
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		base.Body(w, data, http.StatusOK)
	}
}
