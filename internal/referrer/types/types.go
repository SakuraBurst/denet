package types

type User struct {
	ID           int
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	UserName     string `json:"user_name"`
	Password     string `json:"-"`
	ReferrerCode string `json:"referrer_code"`
	Balance      int    `json:"balance"`
}

type FullUser struct {
	ID             int
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	UserName       string  `json:"user_name"`
	Password       string  `json:"-"`
	ReferrerCode   string  `json:"referrer_code"`
	Balance        int     `json:"balance"`
	CompletedTasks []*Task `json:"completed_tasks"`
}

type UserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserName  string `json:"user_name"`
	Password  string `json:"password"`
}

type CompleteTaskRequest struct {
	TaskId int `json:"task_id"`
}

type ReferrerRequest struct {
	ReferrerCode string `json:"referrer_code"`
}

type Task struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Reward      int    `json:"reward"`
}
