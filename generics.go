package grest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
)

type Generic interface {
	Init(cxt *Context, value interface{})
	FindFilter(request *restful.Request, response *restful.Response)
	SaveOne(request *restful.Request, response *restful.Response)
	DeleteOne(request *restful.Request, response *restful.Response)
	ReplaceOne(request *restful.Request, response *restful.Response)
	UpdateOne(request *restful.Request, response *restful.Response)

	WebService(urlPath string)
}

type FilterFunction func()

type GenericAPIView struct {
	APIView
	cxt              *Context
	WS               *restful.WebService
	Value            interface{}
	NewStruct        interface{}
	NewSlice         interface{}
	containerFilters FilterFunction
}

// init cxt model
func (g *GenericAPIView) Init(cxt *Context, value interface{}) {
	g.cxt = cxt
	g.WS = new(restful.WebService)
	g.Value = value

	if value != nil {
		// NewStruct initialize a struct for the Resource
		g.NewStruct = reflect.New(Indirect(reflect.ValueOf(value)).Type()).Interface()

		// NewSlice initialize a slice of struct for the Resource
		sliceType := reflect.SliceOf(reflect.TypeOf(value))
		slice := reflect.MakeSlice(sliceType, 0, 0)
		slicePtr := reflect.New(sliceType)
		slicePtr.Elem().Set(slice)
		g.NewSlice = slicePtr.Interface()
	}
}

func (g *GenericAPIView) WebService(urlPath string) {
	if g.WS == nil {
		g.WS = new(restful.WebService)
	}
	g.WS.Path(fmt.Sprintf("/%s", urlPath)).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	tags := []string{reflect.TypeOf(g.Value).Name()}
	g.WS.Route(g.WS.GET("").To(g.FindFilter).
		Param(g.WS.QueryParameter("filter", `Filter defining withCount, include, fields, where, order, offset, and limit - must be a JSON-encoded string ({"something":"value"})`).DataType("string").Required(false)).
		Doc("query filter").Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "query success", g.NewSlice))

	g.WS.Route(g.WS.POST("").To(g.SaveOne).
		Reads(g.Value, "model").
		Doc("save").Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "save success", g.NewStruct))

	g.WS.Route(g.WS.DELETE("").To(g.DeleteOne).
		Reads(g.Value, "model").
		Doc("delete").Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "delete success", g.NewStruct))

	g.WS.Route(g.WS.PUT("").To(g.ReplaceOne).
		Reads(g.Value, "model").
		Doc("replace").Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "replace success", g.NewStruct))

	g.WS.Route(g.WS.PATCH("").To(g.UpdateOne).
		Reads(g.Value, "model").
		Doc("update").Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(http.StatusOK, "update success", g.NewStruct))

}

// FindFilter adds a request function to handle GET request.
func (g *GenericAPIView) FindFilter(request *restful.Request, response *restful.Response) {
	//http.Error(g.cxt.Response, "Method Not Allowed", 405)
	filter := strings.TrimSpace(request.QueryParameter("filter"))
	filterMap := make(map[string]interface{})
	if filter != "" {
		err := json.Unmarshal([]byte(filter), &filterMap)
		if err != nil {
			response.WriteHeaderAndEntity(http.StatusBadRequest, NewErrorMsg(http.StatusBadRequest, "query data", err.Error()))
			return
		}
	}
	sliceType := reflect.SliceOf(reflect.TypeOf(g.Value))
	slice := reflect.MakeSlice(sliceType, 0, 0)
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(slice)
	results := slicePtr.Interface()
	count, err := g.FindMany(results, filterMap, g.cxt)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "query data", err.Error()))
		return
	}
	response.AddHeader("count", strconv.Itoa(count))
	response.WriteAsJson(results)
}

// SaveOne adds a request function to handle POST request.
func (g *GenericAPIView) SaveOne(request *restful.Request, response *restful.Response) {
	//http.Error(g.cxt.Response, "Method Not Allowed", 405)
	result := reflect.New(Indirect(reflect.ValueOf(g.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "save data", err.Error()))
		return
	}
	err = g.Save(result, g.cxt)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "save data", err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// DeleteOne adds a request function to handle DELETE request.
func (g *GenericAPIView) DeleteOne(request *restful.Request, response *restful.Response) {
	//http.Error(g.cxt.Response, "Method Not Allowed", 405)
	//result := g.NewStruct
	result := reflect.New(Indirect(reflect.ValueOf(g.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "delete data", err.Error()))
		return
	}
	err = g.Delete(result, g.cxt)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "delete data", err.Error()))
		return
	}
	response.WriteAsJson(NewDeleteMsg(1))
}

// ReplaceOne adds a request function to handle PUT request.
func (g *GenericAPIView) ReplaceOne(request *restful.Request, response *restful.Response) {
	//http.Error(g.cxt.Response, "Method Not Allowed", 405)
	//result := g.NewStruct
	result := reflect.New(Indirect(reflect.ValueOf(g.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "replace data", err.Error()))
		return
	}
	err = g.Save(result, g.cxt)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "replace data", err.Error()))
		return
	}
	response.WriteAsJson(result)
}

// UpdateOne adds a request function to handle PATCH request.
func (g *GenericAPIView) UpdateOne(request *restful.Request, response *restful.Response) {
	//http.Error(g.cxt.Response, "Method Not Allowed", 405)
	//result := g.NewStruct
	result := reflect.New(Indirect(reflect.ValueOf(g.Value)).Type()).Interface()
	err := request.ReadEntity(result)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "update data", err.Error()))
		return
	}
	err = g.Save(result, g.cxt)
	if err != nil {
		response.WriteHeaderAndEntity(http.StatusInternalServerError, NewErrorMsg(http.StatusInternalServerError, "update data", err.Error()))
		return
	}
	response.WriteAsJson(result)
}
