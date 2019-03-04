package main

import (
	"log"
	"net"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"iot.tmis.top/common"
)

func main() {
	db, err := gorm.Open("postgres", "host=iot.tmis.top port=5432 user=postgres dbname=iot password=dell@123 sslmode=disable")
	if err != nil {
		log.Panic(err)
	}
	root, _ := os.Getwd()

	// User 用户
	type User struct {
		ID   uint   `json:"id" gorm:"primary_key;AUTO_INCREMENT"`
		Name string `json:"name,omitempty"`
		Age  uint   `json:"age,omitempty"`
	}

	// p := common.NewGenericAPIView("users", User{}, db).DefaultWS()
	// p.RouteBuilder(p.GetWS().GET("/test").To(p.SaveOne).
	// 	Reads(p.Value, "模型").
	// 	Doc("测试").Metadata(restfulspec.KeyOpenAPITags, p.GetTags()).
	// 	Returns(http.StatusOK, "成功", User{}))
	// restful.Add(p.GetWS())

	p := common.NewGenericAPIView("users", User{}, db).DefaultWS()
	restful.Add(p.GetWS())

	// 服务器地址
	serviceURL := net.JoinHostPort("127.0.0.1", "8080")
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"token", "count", "content-disposition"},
		AllowedHeaders: []string{"token", "count", restful.HEADER_ContentType, restful.HEADER_AccessControlAllowOrigin},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "PATCH", "OPTIONS"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer}

	common.NewGenericAPIService(serviceURL, root+"/swagger").
		DefaultConfig().
		Filter(restful.OPTIONSFilter(), cors.Filter).
		Tags(p.Value, "用户").Start()
}
