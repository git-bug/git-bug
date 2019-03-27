package entity

type Interface interface {
	// Id return the Entity identifier
	Id() string
	// HumanId return the Entity identifier truncated for human consumption
	HumanId() string
}
