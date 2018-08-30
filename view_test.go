package grest_test

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/zhgqiang/grest"
)

type User struct {
	grest.APIView
	ID        uint      `json:"id" gorm:"primary_key"`
	Name      string    `json:"name" gorm:"type:varchar(100)"`
	Age       int       `json:"age" gorm:"type:int(3)"`
	Birthday  time.Time `json:"birthday" gorm:"type:varchar(200)"`
	CompanyId uint      `json:"companyId"`
}

func (User) TableName() string {
	return "t_user"
}

type Company struct {
	grest.APIView
	ID          uint   `json:"id" gorm:"primary_key"`
	CompanyName string `json:"companyName" gorm:"type:varchar(100);column:company_name"`
	Users       []User `json:"users" gorm:"ForeignKey:CompanyId"`
}

func (Company) TableName() string {
	return "t_company"
}

func TGormDB() *gorm.DB {
	root := os.Getenv("GOPATH")
	db, err := gorm.Open("sqlite3", path.Join(root, "src/etstone.cn/etrest/testdata/data.db"))
	if err != nil {
		log.Panic(err)
	}
	return db
}

func TContext() *grest.Context {
	return (&grest.Context{}).SetDB(TGormDB())
}

func TSave(cxt *grest.Context, t *testing.T) *User {

	user := new(User)

	u := &User{Name: "test16", Age: 16, Birthday: time.Now(), CompanyId: 1}
	err := user.Save(u, cxt)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestAPIView_Delete(t *testing.T) {
	cxt := TContext()
	user := new(User)

	u := TSave(cxt, t)
	b, _ := json.Marshal(u)
	t.Logf("save %s", string(b))
	//cxt.ResourceID = strconv.Itoa(int(u.ID))
	err := user.Delete(u, cxt)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIView_FindMany_2(t *testing.T) {
	cxt := TContext()
	user := new(User)

	// filter := map[string]interface{}{
	// 	"withCount": true,
	// 	//"include":   "Users",
	// 	//"fields":    []interface{}{"id", "companyName"},
	// 	//"where":     []interface{}{"id=1"},
	// 	"order":  "name",
	// 	"offset": 2,
	// 	"limit":  2,
	// }

	filter := map[string]interface{}{
		"fields": []interface{}{"name"},
	}
	users := new([]User)
	s, err := user.FindMany_2(users, filter, cxt)
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("count:%d,data:%+v", s, *users)
	b, _ := json.Marshal(users)
	t.Logf("find count:%d,data:%+v", s, string(b))
}

func TestAPIView_FindMany(t *testing.T) {
	cxt := TContext()
	user := new(User)

	filter := &grest.Filter{WithCount: true, Order: "name", Offset: "2", Limit: "2"}
	// ws := make([]interface{}, 0)
	// filter := &grest.Filter{WithCount: true, Where: ws, Order: ""}
	// filter := &grest.Filter{Fields: []string{"name"}}
	users := new([]User)
	s, err := user.FindMany(users, filter, cxt)
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("count:%d,data:%+v", s, *users)
	b, _ := json.Marshal(users)
	t.Logf("find count:%d,data:%+v", s, string(b))
}

func TestAPIView_FindMany_3(t *testing.T) {
	cxt := TContext()
	user := new(User)

	filter := &grest.Filter{Fields: []string{"name"}}
	users := new([]User)
	s, err := user.FindMany(users, filter, cxt)
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("count:%d,data:%+v", s, *users)
	b, _ := json.Marshal(users)
	t.Logf("find count:%d,data:%+v", s, string(b))
}

func TestAPIView_FindOne(t *testing.T) {
	cxt := TContext()
	user := new(User)

	u := &User{ID: 16}
	cxt.ResourceID = strconv.Itoa(int(u.ID))
	err := user.FindOne(u, cxt)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(u)
	t.Logf("findone,%+v", string(b))
}
