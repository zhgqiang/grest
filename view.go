// Package common 如果使用update操作,字段必须添加 omitempty.
package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/jinzhu/gorm"
)

// Query 查询条件
type Query map[string]interface{}

// View 模型视图基础接口
type View interface {
	FindMany(interface{}, interface{}, Query, *gorm.DB) error
	Save(interface{}, *gorm.DB) error
	Update(interface{}, *gorm.DB) error
	FindByID(interface{}, interface{}, *gorm.DB) error
	DeleteByID(interface{}, interface{}, *gorm.DB) error
	Delete(interface{}, *gorm.DB) error
}

// APIView 模型视图层
type APIView struct{}

// FindMany 根据查询条件查询数据
// count 查询数据量
func (p *APIView) FindMany(count, result interface{}, query Query, db *gorm.DB) error {
	db = db.Begin()
	// 查询条件
	if query != nil {
		// 查询字段
		if fields, ok := query["fields"]; ok {
			db = db.Select(fields)
		}

		if joins, ok := query["joins"]; ok {
			if j, ok := joins.([]string); ok {
				for _, join := range j {
					db = db.Joins(join)
				}
			} else {
				return errors.New("joins格式不正确")
			}
		}

		if groups, ok := query["groups"]; ok {
			if g, ok := groups.([]string); ok {
				for _, group := range g {
					db = db.Group(group)
				}
			} else {
				return errors.New("groups格式不正确")
			}
		}

		if preloads, ok := query["preloads"]; ok {
			if p, ok := preloads.([]string); ok {
				if len(p) == 1 {
					db = db.Preload(p[0])
				} else if len(p) > 1 {
					db = db.Preload(p[0], p[1:])
				}
			} else {
				return errors.New("preloads格式不正确")
			}
		}

		// where 条件
		if where, ok := query["where"]; ok {
			if w, ok := where.([]interface{}); ok {
				if len(w) == 1 {
					db = db.Where(w[0])
				} else if len(w) > 1 {
					db = db.Where(w[0], w[1:]...)
				}
			} else {
				return errors.New("Where条件格式不正确")
			}
		}

		if withCount, ok := query["withCount"]; ok {
			if w, ok := withCount.(bool); ok {
				if w {
					db = db.Count(&count)
				}
			} else {
				return errors.New("withCount格式不正确")
			}
		}
		// 排序
		if order, ok := query["order"]; ok {
			db = db.Order(order)
		}

		if offset, ok := query["offset"]; ok {
			db = db.Offset(offset)
		}

		if limit, ok := query["limit"]; ok {
			db = db.Limit(limit)
		}
	}
	if db = db.Find(result).Commit(); db.Error != nil {
		return db.Error
	}
	return nil
}

// Save 保存一个数据库模型
func (p *APIView) Save(result interface{}, db *gorm.DB) error {
	if db.NewScope(result).PrimaryKeyZero() {
		return db.Create(result).Error
	}
	return db.Save(result).Error
}

// Update 更新一个数据库模型
func (p *APIView) Update(result interface{}, db *gorm.DB) error {
	return db.Model(result).Updates(result).Error
}

// FindByID 根据id查询数据
func (p *APIView) FindByID(result, id interface{}, db *gorm.DB) error {
	return db.First(result, fmt.Sprintf("%s=?", db.NewScope(result).PrimaryField().DBName), id).Error
}

// DeleteByID 根据ID删除数据
func (p *APIView) DeleteByID(id, result interface{}, db *gorm.DB) error {
	if !db.Find(result).RecordNotFound() {
		return db.Delete(result, fmt.Sprintf("%s=?", db.NewScope(result).PrimaryField().DBName), id).Error
	}
	return gorm.ErrRecordNotFound
}

// Delete 删除数据
func (p *APIView) Delete(result interface{}, db *gorm.DB) error {
	if !db.Find(result).RecordNotFound() {
		return db.Delete(result).Error
	}
	return gorm.ErrRecordNotFound
}

// queryFunction 过滤函数
type queryFunction func()

// GenericAPIView 通用服务创建
type GenericAPIView struct {
	APIView
	db              *gorm.DB
	ws              *restful.WebService
	containerquerys queryFunction
	urlPath         string
	tags            []string

	Value     interface{}
	NewStruct interface{}
	NewSlice  interface{}
}

// NewGenericAPIView 创建通用模型
func NewGenericAPIView(urlPath string, value interface{}, db *gorm.DB) *GenericAPIView {
	p := new(GenericAPIView)
	p.db = db
	p.urlPath = urlPath
	p.ws = new(restful.WebService)
	p.Value = value
	p.tags = []string{reflect.TypeOf(value).Name()}
	// NewStruct initialize a struct for the Resource
	p.NewStruct = reflect.New(Indirect(reflect.ValueOf(value)).Type()).Interface()
	// NewSlice initialize a slice of struct for the Resource
	sliceType := reflect.SliceOf(reflect.TypeOf(value))
	slice := reflect.MakeSlice(sliceType, 0, 0)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	p.NewSlice = slicePtr.Interface()
	return p
}

// DeleteResult 删除结果
type DeleteResult struct {
	Count int `json:"count"`
}

// DefaultWS 默认服务
func (p *GenericAPIView) DefaultWS() *GenericAPIView {
	p.GetWS().
		Path(fmt.Sprintf("/%s", p.urlPath)).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Route(p.GetWS().POST("").To(p.saveOne).
			Reads(p.Value, "模型").
			Doc("保存数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "保存成功", p.NewStruct)).
		Route(p.GetWS().DELETE("").To(p.deleteOne).
			Reads(p.Value, "模型").
			Doc("删除数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "删除成功", DeleteResult{})).
		Route(p.GetWS().PUT("").To(p.replaceOne).
			Reads(p.Value, "模型").
			Doc("替换数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "替换成功", p.NewStruct)).
		Route(p.GetWS().PATCH("").To(p.updateOne).
			Reads(p.Value, "模型").
			Doc("更新数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "更新成功", p.NewStruct)).
		Route(p.GetWS().GET("").To(p.findQuery).
			Param(p.GetWS().QueryParameter("query", `query defining withCount, fields, where, order, offset, and limit - must be a JSON-encoded string ({"something":"value"})`).DataType("string").Required(false)).
			Doc("查询数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "查询成功", p.NewSlice)).
		Route(p.GetWS().DELETE("/{id}").To(p.deleteByID).
			Param(p.GetWS().PathParameter("id", `数据唯一标识`).DataType("integer").Required(true)).
			Doc("根据id删除数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "删除成功", DeleteResult{})).
		Route(p.GetWS().PUT("/{id}").To(p.replaceByID).
			Param(p.GetWS().PathParameter("id", `数据唯一标识`).DataType("integer").Required(true)).
			Reads(p.Value, "模型").
			Doc("根据id删除数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "删除成功", DeleteResult{})).
		Route(p.GetWS().PATCH("/{id}").To(p.updateByID).
			Param(p.GetWS().PathParameter("id", `数据唯一标识`).DataType("integer").Required(true)).
			Reads(p.Value, "模型").
			Doc("根据id更新数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "更新成功", p.NewStruct)).
		Route(p.GetWS().GET("/{id}").To(p.findByID).
			Param(p.GetWS().PathParameter("id", `数据唯一标识`).DataType("integer").Required(true)).
			Doc("根据id查询数据").Metadata(restfulspec.KeyOpenAPITags, p.tags).
			Returns(http.StatusOK, "查询成功", p.NewSlice))

	return p
}

// GetTags 获取tag
func (p *GenericAPIView) GetTags() []string {
	return p.tags
}

// WS 添加服务
func (p *GenericAPIView) WS(ws *restful.WebService) *GenericAPIView {
	p.ws = ws
	return p
}

// RouteBuilder 服务中添加自定义接口
func (p *GenericAPIView) RouteBuilder(builder *restful.RouteBuilder) *GenericAPIView {
	p.GetWS().Route(builder)
	return p
}

// GetWS 获取服务
func (p *GenericAPIView) GetWS() *restful.WebService {
	return p.ws
}

// saveOne 保存一条数据到数据库.
func (p *GenericAPIView) saveOne(request *restful.Request, response *restful.Response) {
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	err = p.Save(result, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}

	response.WriteAsJson(result)
}

// deleteByID 根据id从数据库删除一条数据.
func (p *GenericAPIView) deleteByID(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")

	err := p.DeleteByID(id, reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface(), p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(DeleteResult{Count: 1})
}

// deleteOne 从数据库删除一条数据.
func (p *GenericAPIView) deleteOne(request *restful.Request, response *restful.Response) {
	//result := g.NewStruct
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	err = p.Delete(result, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(DeleteResult{Count: 1})
}

// replaceByID 根据id替换一条数据.
func (p *GenericAPIView) replaceByID(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("id"))
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err = request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	p.db.NewScope(result).PrimaryField().Set(id)
	err = p.Save(result, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// replaceOne 替换数据.
func (p *GenericAPIView) replaceOne(request *restful.Request, response *restful.Response) {
	//result := g.NewStruct
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	err = p.Save(result, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// updateByID 根据id更新数据.
func (p *GenericAPIView) updateByID(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("id"))
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err = request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	p.db.NewScope(result).PrimaryField().Set(id)
	err = p.Update(result, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// updateOne 更新数据.
func (p *GenericAPIView) updateOne(request *restful.Request, response *restful.Response) {
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}

	err = p.Update(result, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// findQuery 根据过滤器查询数据.
func (p *GenericAPIView) findQuery(request *restful.Request, response *restful.Response) {
	query := strings.TrimSpace(request.QueryParameter("query"))
	queryMap := new(Query)
	if query != "" {
		err := json.Unmarshal([]byte(query), queryMap)
		if err != nil {
			response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
			return
		}
	}
	sliceType := reflect.SliceOf(reflect.TypeOf(p.Value))
	slice := reflect.MakeSlice(sliceType, 0, 0)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	results := slicePtr.Interface()
	count := 0
	err := p.FindMany(&count, results, *queryMap, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.AddHeader("count", strconv.Itoa(count))
	response.WriteAsJson(results)
}

// findByID 根据id查询数据
func (p *GenericAPIView) findByID(request *restful.Request, response *restful.Response) {
	id, err := strconv.Atoi(request.PathParameter("id"))
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, NewResponseMsg(err.Error()))
		return
	}
	result := reflect.New(Indirect(reflect.ValueOf(p.Value)).Type()).Interface()
	err = p.FindByID(result, id, p.db)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewResponseMsg(err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// GenericAPIService 创建服务
type GenericAPIService struct {
	tags                   []spec.Tag
	config                 restfulspec.Config
	serviceURL, swaggerDir string
}

// NewGenericAPIService 创建服务
func NewGenericAPIService(serviceURL, swaggerDir string) *GenericAPIService {
	return &GenericAPIService{tags: make([]spec.Tag, 0), serviceURL: serviceURL, swaggerDir: swaggerDir}
}

// Filter 服务中添加过滤器
func (p *GenericAPIService) Filter(filters ...restful.FilterFunction) *GenericAPIService {
	for _, filter := range filters {
		restful.Filter(filter)
	}
	return p
}

// CreateTags 创建服务文档tag集合
func (p *GenericAPIService) CreateTags(tags []spec.Tag) *GenericAPIService {
	p.tags = tags
	return p
}

// GetTags 创建服务文档tag集合
func (p *GenericAPIService) GetTags() []spec.Tag {
	return p.tags
}

// Tags 添加tag
func (p *GenericAPIService) Tags(value interface{}, desc string) *GenericAPIService {
	p.tags = append(p.tags, spec.Tag{
		TagProps: spec.TagProps{
			Name:        reflect.TypeOf(value).Name(),
			Description: desc,
		},
	})
	return p
}

// DefaultConfig 根据默认配置创建文档
func (p *GenericAPIService) DefaultConfig() *GenericAPIService {
	p.config = restfulspec.Config{
		WebServices:    restful.RegisteredWebServices(),
		WebServicesURL: p.serviceURL,
		APIPath:        "/apidocs.json",
		PostBuildSwaggerObjectHandler: func(swagger *spec.Swagger) {
			swagger.Tags = p.tags
		},
	}
	return p
}

// Config 添加接口配置
func (p *GenericAPIService) Config(config restfulspec.Config) *GenericAPIService {
	p.config = config
	return p
}

// GetConfig 获取接口配置
func (p *GenericAPIService) GetConfig() restfulspec.Config {
	return p.config
}

// Start 启动服务
func (p *GenericAPIService) Start() {
	restful.Add(restfulspec.NewOpenAPIService(p.config))
	http.Handle("/apidocs/", http.StripPrefix("/apidocs/", http.FileServer(http.Dir(p.swaggerDir))))
	log.Printf("接口已启动 http://%s/apidocs \n", p.serviceURL)
	log.Panic(http.ListenAndServe(p.serviceURL, nil))
}

// Stop 停止服务
func (p *GenericAPIService) Stop() {

}
