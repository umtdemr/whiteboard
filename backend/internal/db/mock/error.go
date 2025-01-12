package mockdb

// MockPgError a mock struct for pgconn.PgError
type MockPgError struct {
	ErrorCode string
}

func (e MockPgError) Error() string {
	return "mock pg error: " + e.ErrorCode
}

func (e MockPgError) Code() string {
	return e.ErrorCode
}
