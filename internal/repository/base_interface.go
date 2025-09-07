package repository

type BaseRepositoryMethods[T any] interface {
	// Create a new record and return the created entity with updated fields (e.g., auto-increment IDs)
	Create(entity *T) (*T, error)

	// Create or Update
	CreateOrUpdate(entity *T) (*T, error)

	// Retrieve all records with optional preloads
	FindAll(preloads ...string) ([]T, error)

	// Retrieve a record by UUID with optional preloads
	FindByUUID(uuid any, preloads ...string) (*T, error)

	// Retrieve multiple records by UUIDs with optional preloads
	FindByUUIDs(uuids []string, preloads ...string) ([]T, error)

	// Retrieve a record by ID with optional preloads
	FindByID(id any, preloads ...string) (*T, error)

	// Update a record by UUID and return the updated entity
	UpdateByUUID(uuid any, updatedData any) (*T, error)

	// Update a record by ID and return the updated entity
	UpdateByID(id any, updatedData any) (*T, error)

	// Delete a record by UUID and return the deleted entity
	DeleteByUUID(uuid any) error

	// Delete a record by ID and return the deleted entity
	DeleteByID(id any) error

	// Paginate through records with optional preloads
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[T], error)
}
