package repository

// TableNames is the set of physical table names. Using a typed enum here means a
// typo in a query is a compile error, not a runtime "relation does not exist".
type TableNames string

const (
	Branches            TableNames = "branches"
	Cars                TableNames = "cars"
	PriceConfigurations TableNames = "price_configurations"
	Users               TableNames = "users"
	Bookings            TableNames = "bookings"
)

func (t TableNames) ToString() string { return string(t) }
