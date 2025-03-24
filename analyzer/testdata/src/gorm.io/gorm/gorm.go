package gorm

type DB struct {
	Error error
}

func (db *DB) Model(value interface{}) *DB {
	return &DB{}
}

func (db *DB) Where(query interface{}, args ...interface{}) *DB {
	return &DB{}
}

func (db *DB) Omit(columns ...string) *DB {
	return &DB{}
}

func (db *DB) Select(query interface{}, args ...interface{}) *DB {
	return &DB{}
}

func (db *DB) Association(column string) *Association {
	return &Association{}
}

func (db *DB) Update(query interface{}, args ...interface{}) *DB {
	return db
}

func (db *DB) Updates(values interface{}) *DB {
	return db
}

type Error struct{}

type Association struct{}

func (a *Association) Replace(values ...interface{}) error {
	return nil
}
