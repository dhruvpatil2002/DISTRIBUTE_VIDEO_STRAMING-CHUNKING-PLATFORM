
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user *models.User,
) error {
	query := `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
	).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
}

func (r *UserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, role,
		       created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, role,
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}


func (r *UserRepository) GetAll(
    ctx context.Context,
    limit int,
    offset int,
) ([]models.User, int64, error) {


     if limit < 1 {
        limit = 20
    }

    if limit > 100 {
        limit = 100
    }

    if offset < 0 {
        offset = 0
    }
    // Fetch only the required users.
    query := `
        SELECT id, name, email, role, created_at, updated_at
        FROM users
        ORDER BY created_at DESC, id DESC
        LIMIT $1 OFFSET $2
    `

    rows, err := r.db.Query(ctx, query, limit, offset)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    users := make([]models.User, 0, limit)

    for rows.Next() {
        var user models.User

        err := rows.Scan(
            &user.ID,
            &user.Name,
            &user.Email,
            &user.Role,
            &user.CreatedAt,
            &user.UpdatedAt,
        )

        if err != nil {
            return nil, 0, err
        }

        users = append(users, user)
    }

    if err := rows.Err(); err != nil {
        return nil, 0, err
    }

    // Total number of users for pagination metadata.
    var total int64

    err = r.db.QueryRow(
        ctx,
        `SELECT COUNT(*) FROM users`,
    ).Scan(&total)

    if err != nil {
        return nil, 0, err
    }

    return users, total, nil
}


func (r *UserRepository) UpdateRole(
    ctx context.Context,
    userID string,
    role string,
) (*models.User, error) {

    query := `
        UPDATE users
        SET role = $1,
            updated_at = NOW()
        WHERE id = $2
        RETURNING
            id,
            name,
            email,
            role,
            created_at,
            updated_at
    `

    user := &models.User{}

    err := r.db.QueryRow(
        ctx,
        query,
        role,
        userID,
    ).Scan(
        &user.ID,
        &user.Name,
        &user.Email,
        &user.Role,
        &user.CreatedAt,
        &user.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return user, nil
}


