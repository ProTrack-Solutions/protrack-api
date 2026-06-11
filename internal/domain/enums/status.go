package enums

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
	StatusDeleted  Status = "DELETED"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusDeleted:
		return true
	default:
		return false
	}
}
