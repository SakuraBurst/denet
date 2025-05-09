package database

import (
	"context"
	"time"

	"github.com/SakuraBurst/denet/internal/referrer/config"
	"github.com/SakuraBurst/denet/internal/referrer/types"
	"github.com/go-faster/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DB struct {
	Conn   *pgxpool.Pool
	logger *zap.Logger
}

func NewDB(cfg *config.Config, logger *zap.Logger) (*DB, error) {
	conn, err := initDatabase(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "InitDatabase failed")
	}
	return &DB{Conn: conn, logger: logger.Named("db")}, nil
}

func initDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cl := context.WithTimeout(context.Background(), 5*time.Second)
	defer cl()
	pool, err := pgxpool.New(ctx, cfg.DataBaseURL)
	if err != nil {

		return nil, errors.Wrap(err, "pgxpool.New failed")
	}
	return pool, nil
}

func (d *DB) CreateNewUser(ctx context.Context, user *types.UserRequest, referrerCode string) error {
	row := d.Conn.QueryRow(ctx, "insert into users (first_name, last_name, user_name, password, balance, referrer_code) values ($1, $2, $3, $4, $5, $6) on conflict (user_name) do nothing returning id", user.FirstName, user.LastName, user.UserName, user.Password, 0, referrerCode)
	var id int
	err := row.Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserAlreadyExist
	}
	if err != nil {
		return errors.Wrap(err, "row.Scan failed: ")
	}
	return nil
}

func (d *DB) GetFullUserInfo(ctx context.Context, userID int) (*types.FullUser, error) {
	row := d.Conn.QueryRow(ctx, "select id, first_name, last_name, user_name, password, balance, referrer_code from users where id = $1", userID)
	user := &types.FullUser{}
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.UserName, &user.Password, &user.Balance, &user.ReferrerCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotExist
	}
	if err != nil {
		return nil, errors.Wrap(err, "row.Scan failed: ")
	}
	rows, err := d.Conn.Query(ctx, "select t2.* from tasks_to_users t1 left join tasks t2 on t2.id = t1.task_id where t1.user_id = $1", userID)
	if err != nil {
		return nil, errors.Wrap(err, "conn.Query failed: ")
	}
	defer rows.Close()
	result, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.Task])
	if err != nil {
		return nil, errors.Wrap(err, "pgx.CollectRows failed: ")
	}
	user.CompletedTasks = result
	return user, nil
}

// GetUserByUserName возвращает весего юзера на всякий случай, вдруг где-то еще пригодиться
func (d *DB) GetUserByUserName(ctx context.Context, userName string) (*types.User, error) {
	row := d.Conn.QueryRow(ctx, "select id, first_name, last_name, user_name, password, balance from users where user_name = $1", userName)
	user := &types.User{}
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.UserName, &user.Password, &user.Balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotExist
	}
	if err != nil {
		return nil, errors.Wrap(err, "row.Scan failed: ")
	}
	return user, nil
}

// GetUserByReferrerCode возвращает весего юзера на всякий случай, вдруг где-то еще пригодиться
func (d *DB) GetUserByReferrerCode(ctx context.Context, referrerCode string) (*types.User, error) {
	row := d.Conn.QueryRow(ctx, "select id, first_name, last_name, user_name, password, balance from users where referrer_code = $1", referrerCode)
	user := &types.User{}
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.UserName, &user.Password, &user.Balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotExist
	}
	if err != nil {
		return nil, errors.Wrap(err, "row.Scan failed: ")
	}
	return user, nil
}

func (d *DB) GetTopUsersByBalance(ctx context.Context, limit int) ([]*types.User, error) {
	rows, err := d.Conn.Query(ctx, "select * from users order by balance limit $1", limit)
	if err != nil {
		return nil, errors.Wrap(err, "Conn.Query failed: ")
	}
	result, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.User])
	if err != nil {
		return nil, errors.Wrap(err, "pgx.CollectRows failed: ")
	}
	return result, nil
}

func (d *DB) CompleteTask(ctx context.Context, taskID, userID int) (int, error) {
	tx, err := d.Conn.Begin(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "conn.Begin failed: ")
	}

	rollback := func() {
		if err := tx.Rollback(ctx); err != nil {
			d.logger.Error("tx.Rollback failed", zap.Error(err))
		}
	}

	var balance int
	row := tx.QueryRow(ctx, "select balance from users where id = $1 for update ", userID)
	err = row.Scan(&balance)
	if err != nil {
		rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUserNotExist
		}
		return 0, err
	}
	var reward int
	row = tx.QueryRow(ctx, "select reward from tasks where id = $1", taskID)
	err = row.Scan(&reward)
	if err != nil {
		rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrTaskNotExist
		}
		return 0, err
	}

	var taskToUserId int
	row = tx.QueryRow(ctx, "insert into tasks_to_users (task_id, user_id) values ($1, $2) on conflict do nothing returning id", taskID, userID)
	err = row.Scan(&taskToUserId)
	if err != nil {
		rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrAlreadyCompletedTask
		}
		return 0, err
	}

	_, err = tx.Exec(ctx, "update users set balance = $2 where id = $1", userID, balance+reward)
	if err != nil {
		rollback()
		return 0, errors.Wrap(err, "tx.Exec failed: ")
	}
	return balance + reward, tx.Commit(ctx)
}

func (d *DB) CreateNewTask(ctx context.Context, task *types.Task) (int, error) {
	row := d.Conn.QueryRow(ctx, "insert into tasks (description, reward) values ($1, $2) on conflict (description) do nothing returning id", task.Description, task.Reward)
	var id int
	err := row.Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrTaskAlreadyExist
	}
	if err != nil {
		return 0, errors.Wrap(err, "row.Scan failed: ")
	}
	return id, nil
}

func (d *DB) GetTaskById(ctx context.Context, taskID int) (*types.Task, error) {
	row := d.Conn.QueryRow(ctx, "select id, description, reward from tasks where id = $1", taskID)
	task := &types.Task{}
	err := row.Scan(&task.ID, &task.Description, &task.Reward)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotExist
	}
	if err != nil {
		return nil, errors.Wrap(err, "row.Scan failed: ")
	}
	return task, nil
}

func (d *DB) GetAllTasks(ctx context.Context) ([]*types.Task, error) {
	rows, err := d.Conn.Query(ctx, "select * from tasks")
	if err != nil {
		return nil, errors.Wrap(err, "Conn.Query failed: ")
	}
	result, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.Task])
	if err != nil {
		return nil, errors.Wrap(err, "pgx.CollectRows failed: ")
	}
	return result, nil
}

func (d *DB) UpdateTaskReward(ctx context.Context, id, newReward int) error {
	_, err := d.Conn.Exec(ctx, "update tasks set reward = $2 where id = $1", id, newReward)
	return err
}

func (d *DB) RewardUser(ctx context.Context, userID, rewardValue int) error {
	_, err := d.Conn.Exec(ctx, "update users set balance = balance + $2 where id = $1", userID, rewardValue)
	return err
}
