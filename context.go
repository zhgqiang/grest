package grest

import (
	"github.com/emicklei/go-restful"
	"github.com/jinzhu/gorm"
)

// Context ...
type Context struct {
	DB         *gorm.DB
	ResourceID string
	Request    *restful.Request
	Response   *restful.Response
}

// Clone clone current context
func (context *Context) Clone() *Context {
	var clone = *context
	return &clone
}

// GetDB get db from current context
func (context *Context) GetDB() *gorm.DB {
	return context.DB
}

// SetDB set db into current context
func (context *Context) SetDB(db *gorm.DB) *Context {
	context.DB = db
	return context
}
