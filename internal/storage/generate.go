package storage

//go:generate sqlc generate
//go:generate mockgen -destination=querier_mock.go -typed=true -package=storage . Querier
