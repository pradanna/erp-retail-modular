package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type mysqlUserRepository struct {
	db *sql.DB
}

// NewUserRepository membuat implementasi baru UserRepository berbasis MySQL.
func NewUserRepository(db *sql.DB) UserRepository {
	return &mysqlUserRepository{db: db}
}

// Save menyimpan pengguna baru ke database.
func (r *mysqlUserRepository) Save(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (
			id, name, username, email, password_hash, role, location_id, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Name, user.Username, user.Email, user.PasswordHash,
		string(user.Role), user.LocationID, user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		var mySQLErr *mysql.MySQLError
		if errors.As(err, &mySQLErr) && mySQLErr.Number == 1062 {
			if strings.Contains(mySQLErr.Message, "username") {
				return ErrUsernameExists
			}
			if strings.Contains(mySQLErr.Message, "email") {
				return ErrEmailExists
			}
		}
		return fmt.Errorf("gagal insert user: %w", err)
	}
	return nil
}

// Update memperbarui data pengguna.
func (r *mysqlUserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users SET
			name = ?, username = ?, email = ?, password_hash = ?, role = ?,
			location_id = ?, is_active = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		user.Name, user.Username, user.Email, user.PasswordHash, string(user.Role),
		user.LocationID, user.IsActive, user.UpdatedAt, user.ID,
	)
	if err != nil {
		var mySQLErr *mysql.MySQLError
		if errors.As(err, &mySQLErr) && mySQLErr.Number == 1062 {
			if strings.Contains(mySQLErr.Message, "username") {
				return ErrUsernameExists
			}
			if strings.Contains(mySQLErr.Message, "email") {
				return ErrEmailExists
			}
		}
		return fmt.Errorf("gagal update user: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal cek rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// FindByID mencari user berdasarkan ID.
func (r *mysqlUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, name, username, email, password_hash, role, location_id, is_active, created_at, updated_at
		FROM users
		WHERE id = ?`

	var u User
	var roleStr string
	var locID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Name, &u.Username, &u.Email, &u.PasswordHash, &roleStr, &locID,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("gagal query user by id: %w", err)
	}
	u.Role = UserRole(roleStr)
	if locID.Valid {
		u.LocationID = &locID.String
	}
	return &u, nil
}

// FindByUsernameOrEmail mencari user berdasarkan username atau alamat email untuk login.
func (r *mysqlUserRepository) FindByUsernameOrEmail(ctx context.Context, identifier string) (*User, error) {
	query := `
		SELECT id, name, username, email, password_hash, role, location_id, is_active, created_at, updated_at
		FROM users
		WHERE username = ? OR email = ?
		LIMIT 1`

	cleanIdentifier := strings.ToLower(strings.TrimSpace(identifier))
	var u User
	var roleStr string
	var locID sql.NullString

	err := r.db.QueryRowContext(ctx, query, cleanIdentifier, cleanIdentifier).Scan(
		&u.ID, &u.Name, &u.Username, &u.Email, &u.PasswordHash, &roleStr, &locID,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("gagal query user by identifier: %w", err)
	}
	u.Role = UserRole(roleStr)
	if locID.Valid {
		u.LocationID = &locID.String
	}
	return &u, nil
}

// List mengambil daftar user dengan filter opsional.
func (r *mysqlUserRepository) List(ctx context.Context, role *UserRole, locationID *string, isActiveOnly bool) ([]*User, error) {
	query := `
		SELECT id, name, username, email, password_hash, role, location_id, is_active, created_at, updated_at
		FROM users
		WHERE 1=1`
	var args []interface{}

	if role != nil {
		query += " AND role = ?"
		args = append(args, string(*role))
	}
	if locationID != nil {
		query += " AND location_id = ?"
		args = append(args, *locationID)
	}
	if isActiveOnly {
		query += " AND is_active = TRUE"
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		var roleStr string
		var locID sql.NullString
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Username, &u.Email, &u.PasswordHash, &roleStr, &locID,
			&u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan user: %w", err)
		}
		u.Role = UserRole(roleStr)
		if locID.Valid {
			u.LocationID = &locID.String
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi users: %w", err)
	}
	return users, nil
}
