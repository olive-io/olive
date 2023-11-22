package backend

type HookFunc func(tx IBatchTx)

// IHooks allow to add additional logic executed during transaction lifetime.
type IHooks interface {
	// OnPreCommitUnsafe is executed before Commit of transactions.
	// The given transaction is already locked.
	OnPreCommitUnsafe(tx IBatchTx)
}

type hooks struct {
	onPreCommitUnsafe HookFunc
}

func (h hooks) OnPreCommitUnsafe(tx IBatchTx) {
	h.onPreCommitUnsafe(tx)
}

func NewHooks(onPreCommitUnsafe HookFunc) IHooks {
	return hooks{onPreCommitUnsafe: onPreCommitUnsafe}
}
