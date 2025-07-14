package repository

type BaseRepositoryMethods[T any] interface {
	Create(entity *T) error
	FindAll(preloads ...string) ([]T, error)
	FindByUUID(uuid any, preloads ...string) (*T, error)
	FindByID(id any, preloads ...string) (*T, error)
	UpdateByUUID(uuid any, updatedData any) error
	UpdateByID(id any, updatedData any) error
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[T], error)
}
