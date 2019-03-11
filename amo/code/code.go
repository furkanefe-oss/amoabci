package code

const (
	TxCodeOK uint32 = iota
	TxCodeBadParam
	TxCodeNotEnoughBalance
	TxCodeSelfTransaction
	TxCodePermissionDenied
	TxCodeTargetAlreadyBought
	TxCodeTargetAlreadyExists
	TxCodeTargetNotExists
	TxCodeBadSignature
	TxCodeRequestNotExists
)

const (
	QueryCodeOK uint32 = iota
	QueryCodeBadPath
	QueryCodeNoKey
	QueryCodeBadKey
	QueryCodeNoMatch
)