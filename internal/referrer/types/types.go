package types

type User struct {
	ID             int
	FirstName      string
	LastName       string
	UserName       string
	Password       []byte
	ReferrerCode   string
	CompletedTasks []Task
}

type Task struct {
	ID          int
	Description string
	Reward      int
}
