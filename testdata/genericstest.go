package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/zhgqiang/grest"
)

type User struct {
	ID        uint      `json:"id" gorm:"primary_key"`
	Name      string    `json:"name" gorm:"type:varchar(100)"`
	Age       int       `json:"age" gorm:"type:int(3)"`
	Birthday  time.Time `json:"birthday" gorm:"type:varchar(200)"`
	CompanyId uint      `json:"companyId"`
}

func (User) TableName() string {
	return "t_user"
}

func main() {
	u := User{}
	root := os.Getenv("GOPATH")
	db, err := gorm.Open("sqlite3", path.Join(root, "src/etstone.cn/etrest/testdata/data.db"))
	if err != nil {
		log.Panic(err)
	}
	g := grest.GenericAPIView{}
	cxt := new(grest.Context).SetDB(db)
	g.Init(cxt, u)
	g.WebService("users")
	restful.Add(g.WS)
	tags := make([]spec.Tag, 0)
	serviceUrl := net.JoinHostPort("", "9000")
	restful.Filter(restful.OPTIONSFilter())
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-Custom-Header"},
		AllowedHeaders: []string{"X-Custom-Header", "X-Additional-Header", restful.HEADER_ContentType, restful.HEADER_AccessControlAllowOrigin},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "PATCH", "OPTIONS"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer}

	restful.Filter(cors.Filter)
	config := restfulspec.Config{
		WebServices:    restful.RegisteredWebServices(),
		WebServicesURL: serviceUrl,
		APIPath:        "/apidocs.json",
		PostBuildSwaggerObjectHandler: func(swagger *spec.Swagger) {
			swagger.Info = &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       "测试",
					Description: "测试文档",
					//Contact:     "",
				},
			}
			swagger.Tags = tags
		},
	}

	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))
	http.Handle("/apidocs/", http.StripPrefix("/apidocs/", http.FileServer(http.Dir(path.Join(grest.AppRoot, "swagger")))))
	log.Printf("接口访问地址: http://%s/apidocs \n", serviceUrl)
	log.Panic(http.ListenAndServe(serviceUrl, nil))
}
