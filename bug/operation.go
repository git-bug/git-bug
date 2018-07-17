package bug

type OperationType int

const (
	_ OperationType = iota
	CreateOp
	SetTitleOp
	AddCommentOp
	SetStatusOp
)

type Operation interface {
	OpType() OperationType
	Apply(snapshot Snapshot) Snapshot
}

type OpBase struct {
	OperationType OperationType
}

func (op OpBase) OpType() OperationType {
	return op.OperationType
}
