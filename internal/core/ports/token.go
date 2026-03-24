package ports

type TokenIssuer interface {
	Issue(userID uint64) (string, error)
}
