package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/neracastle/go-libs/pkg/db"
	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"

	domain "github.com/neracastle/auth/internal/domain/user"
	"github.com/neracastle/auth/internal/repository/user"
	pgmodel "github.com/neracastle/auth/internal/repository/user/postgres/model"
)

const (
	idColumn       = "id"
	emailColumn    = "email"
	passwordColumn = "password"
	nameColumn     = "name"
	roleColumn     = "role"
	createdColumn  = "created_at"
	updateColumn   = "updated_at"
)

const (
	saveMethod   = "repository.user.postgres.Save"
	updateMethod = "repository.user.postgres.Update"
	deleteMethod = "repository.user.postgres.Delete"
	getMethod    = "repository.user.postgres.Get"
)

var _ user.Repository = (*repo)(nil)

type repo struct {
	conn db.Client
}

// New новый экземпляр репозитория pg
func New(conn db.Client) user.Repository {
	instance := &repo{conn: conn}

	return instance
}

func (r *repo) Save(ctx context.Context, user *domain.User) error {
	log := logger.GetLogger(ctx).With(slog.String("method", saveMethod))
	dto := FromDomainToRepo(user)

	pwdHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash password", slog.String("error", err.Error()))
		return err
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query, args, err := psql.Insert("auth.users").
		Columns(emailColumn, passwordColumn, nameColumn, roleColumn).
		Values(dto.Email, pwdHash, dto.Name, dto.IsAdmin).
		Suffix(fmt.Sprintf("RETURNING %s", idColumn)).
		ToSql()
	if err != nil {
		log.Error("failed to build update query", slog.String("error", err.Error()))
		return err
	}

	q := db.Query{Name: saveMethod, QueryRaw: query}
	err = r.conn.DB().QueryRow(ctx, q, args...).Scan(&user.ID)
	if err != nil {
		log.Error("failed to save user in db", slog.String("error", err.Error()))
		return err
	}

	log.Debug("saved user in db", slog.Int64("id", user.ID))

	return nil
}

func (r *repo) Update(ctx context.Context, user *domain.User) error {
	log := logger.GetLogger(ctx).With(slog.String("method", updateMethod))
	dto := FromDomainToRepo(user)

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query, args, err := psql.Update("auth.users").
		Set(emailColumn, dto.Email).
		Set(nameColumn, dto.Name).
		Set(passwordColumn, dto.Password).
		Set(roleColumn, dto.IsAdmin).
		Set(updateColumn, sq.Expr("now()")).
		Where(sq.Eq{idColumn: dto.ID}).
		ToSql()
	if err != nil {
		log.Error("failed to build update query", slog.String("error", err.Error()))
		return err
	}

	q := db.Query{Name: updateMethod, QueryRaw: query}
	_, err = r.conn.DB().Exec(ctx, q, args...)
	if err != nil {
		log.Error("failed to update user in db", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (r *repo) Delete(ctx context.Context, id int64) error {
	log := logger.GetLogger(ctx).With(slog.String("method", deleteMethod))
	q := db.Query{Name: deleteMethod, QueryRaw: "DELETE FROM auth.users WHERE id = $1"}
	qr, err := r.conn.DB().Exec(ctx, q, id)
	if err != nil {
		log.Error("failed to delete user", slog.String("error", err.Error()))
		return err
	}

	if qr.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *repo) Get(ctx context.Context, filter user.SearchFilter) (*domain.User, error) {
	log := logger.GetLogger(ctx).With(slog.String("method", getMethod), slog.Int64("user_id", filter.ID), slog.String("email", filter.Email))

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	selQuery := psql.Select(idColumn, emailColumn, passwordColumn, nameColumn, roleColumn, createdColumn).From("auth.users")

	if filter.ID > 0 {
		selQuery = selQuery.Where(sq.Eq{idColumn: filter.ID})
	}

	if filter.Email != "" {
		selQuery = selQuery.Where(sq.Eq{emailColumn: filter.Email})
	}

	queryStr, args, err := selQuery.ToSql()
	if err != nil {
		return nil, err
	}

	q := db.Query{Name: getMethod, QueryRaw: queryStr}
	res, err := r.conn.DB().Query(ctx, q, args...)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		log.Error("failed to get user from db", slog.String("error", err.Error()))

		return nil, err
	}

	dto, err := pgx.CollectOneRow(res, pgx.RowToStructByName[pgmodel.UserDTO])
	if err != nil {
		return nil, err
	}

	userAggr := FromRepoToDomain(dto)

	return userAggr, nil
}
