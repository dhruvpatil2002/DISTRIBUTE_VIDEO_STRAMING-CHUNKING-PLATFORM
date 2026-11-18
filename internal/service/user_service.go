package service

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
"crypto/subtle"
    "os"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/repository"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserNotFound       = errors.New("user not found")
	ErrAdminRegistrationDenied = errors.New(
    "admin registration not authorized",
)
)

type UserService struct {
	repo      *repository.UserRepository
	jwtSecret []byte
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Role      string
    AdminCode string
}
	


type LoginInput struct {
	Email    string
	Password string
	
}

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        *models.User `json:"user"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewUserService(
	repo *repository.UserRepository,
	jwtSecret string,
) *UserService {
	return &UserService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *UserService) Register(
	ctx context.Context,
	input RegisterInput,
) (*models.User, error) {

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Role = strings.ToLower(strings.TrimSpace(input.Role))

	if input.Name == "" ||
		!strings.Contains(input.Email, "@") ||
		len(input.Password) < 8 {
		return nil, ErrInvalidInput
	}

	existing, err := s.repo.GetByEmail(ctx, input.Email)

	if err == nil && existing != nil {
		return nil, ErrUserExists
	}

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}
	if input.Role == "" {
		input.Role = models.RoleUser
	}
	if !models.IsValidRole(input.Role) {
		return nil, ErrInvalidInput
	}
	if input.Role == models.RoleAdmin {
		expectedCode := os.Getenv("ADMIN_REGISTRATION_CODE")

if expectedCode == "" ||
			subtle.ConstantTimeCompare(
				[]byte(input.AdminCode),
				[]byte(expectedCode),
			) != 1 {
			return nil, ErrAdminRegistrationDenied
		}
	}

	
	

	user := &models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(passwordHash),
		Role:     input.Role, // Default role is "user"
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	user.Password = ""

	return user, nil
}

func (s *UserService) Login(
	ctx context.Context,
	input LoginInput,
) (*AuthResponse, error) {

	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.repo.GetByEmail(ctx, email)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}

	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(input.Password),
	)

	if err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	tokenExpiry := now.Add(15 * time.Minute)

	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(tokenExpiry),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	user.Password = ""

	return &AuthResponse{
		AccessToken: signedToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(tokenExpiry).Seconds()),
		User:        user,
	}, nil
}

func (s *UserService) GetUserFromToken(
	ctx context.Context,
	tokenString string,
) (*models.User, error) {

	if len(s.jwtSecret) == 0 {
		return nil, errors.New("JWT secret is empty")
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {

			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return s.jwtSecret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.UserID == "" {
		return nil, errors.New("user ID missing from JWT claims")
	}

	user, err := s.repo.GetByID(ctx, claims.UserID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	user.Password = ""

	return user, nil
}

func (s *UserService) GetAllUsers(
	ctx context.Context,
	page int,
	limit int,
) ([]models.User, int64, error) {

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 20
	}

	// Protect the database from excessively large requests.
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	return s.repo.GetAll(ctx, limit, offset)
}
func (s *UserService) UpdateUserRole(
    ctx context.Context,
    userID string,
    role string,
) (*models.User, error) {

    if !models.IsValidRole(role) {
        return nil, ErrInvalidInput
    }

    user, err := s.repo.UpdateRole(
        ctx,
        userID,
        role,
    )

    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrUserNotFound
    }

    if err != nil {
        return nil, err
    }

    return user, nil
}



func (s *UserService) GetUserByID(
    ctx context.Context,
    userID string,
) (*models.User, error) {

    user, err := s.repo.GetByID(ctx, userID)

    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrUserNotFound
    }

    if err != nil {
        return nil, err
    }

    user.Password = ""

    return user, nil
}