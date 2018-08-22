package grest

type View interface {
	FindMany(interface{}, map[string]interface{}, *Context) (int, error)
	Save(interface{}, *Context) error
	FindOne(interface{}, *Context) error
	Delete(interface{}, *Context) error
}
