package repository

// TableNames is the enum of physical table names. Querying through it instead of
// string literals turns a typo into a compile error.
type TableNames string

const (
	Users         TableNames = "users"
	Expenses      TableNames = "expenses"
	ExpenseSplits TableNames = "expense_splits"
)

func (t TableNames) ToString() string { return string(t) }
