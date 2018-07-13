package bug

type OperationType int

const (
	UNKNOW OperationType = iota
	CREATE
	SET_TITLE
	ADD_COMMENT
)

type Operation interface {
	OpType() OperationType
	Apply(snapshot Snapshot) Snapshot
}

type OpBase struct {
	OperationType OperationType `json:"op"`
}

func (op OpBase) OpType() OperationType {
	return op.OperationType
}
