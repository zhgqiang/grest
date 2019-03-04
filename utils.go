package common

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/now"
	"github.com/microcosm-cc/bluemonday"
)

// AppRoot app root path
var AppRoot, _ = os.Getwd()

// ContextKey defined type used for context's key
type ContextKey string

// ContextDBName db name used for context
var ContextDBName ContextKey = "ContextDB"

// HTMLSanitizer html sanitizer to avoid XSS
var HTMLSanitizer = bluemonday.UGCPolicy()

func init() {
	HTMLSanitizer.AllowStandardAttributes()
	if path := os.Getenv("WEB_ROOT"); path != "" {
		AppRoot = path
	}
}

// GOPATH return GOPATH from env
func GOPATH() []string {
	paths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	if len(paths) == 0 {
		fmt.Println("GOPATH doesn't exist")
	}
	return paths
}

// GetDBFromRequest get database from request
var GetDBFromRequest = func(req *http.Request) *gorm.DB {
	db := req.Context().Value(ContextDBName)
	if tx, ok := db.(*gorm.DB); ok {
		return tx
	}

	return nil
}

// HumanizeString Humanize separates string based on capitalizd letters
// e.g. "OrderItem" -> "Order Item"
func HumanizeString(str string) string {
	var human []rune
	for i, l := range str {
		if i > 0 && isUppercase(byte(l)) {
			if (!isUppercase(str[i-1]) && str[i-1] != ' ') || (i+1 < len(str) && !isUppercase(str[i+1]) && str[i+1] != ' ' && str[i-1] != ' ') {
				human = append(human, rune(' '))
			}
		}
		human = append(human, l)
	}
	return strings.Title(string(human))
}

// FirstCharToLow e.g. "Order" -> "order"
func FirstCharToLow(str string) string {
	b := bytes.Buffer{}
	for i, l := range str {
		if i == 0 && ('A' <= l && l <= 'Z') {
			b.WriteByte(byte(l) + 32)
		} else {
			b.WriteByte(byte(l))
		}
	}
	return b.String()
}

func isUppercase(char byte) bool {
	return 'A' <= char && char <= 'Z'
}

var asicsiiRegexp = regexp.MustCompile("^(\\w|\\s|-|!)*$")

// ToParamString replaces spaces and separates words (by uppercase letters) with
// underscores in a string, also downcase it
// e.g. ToParamString -> to_param_string, To ParamString -> to_param_string
func ToParamString(str string) string {
	if asicsiiRegexp.MatchString(str) {
		return gorm.ToDBName(strings.Replace(str, " ", "_", -1))
	}
	return slug.Make(str)
}

// PatchURL updates the query part of the request url.
//     PatchURL("google.com","key","value") => "google.com?key=value"
func PatchURL(originalURL string, params ...interface{}) (patchedURL string, err error) {
	url, err := url.Parse(originalURL)
	if err != nil {
		return
	}

	query := url.Query()
	for i := 0; i < len(params)/2; i++ {
		// Check if params is key&value pair
		key := fmt.Sprintf("%v", params[i*2])
		value := fmt.Sprintf("%v", params[i*2+1])

		if value == "" {
			query.Del(key)
		} else {
			query.Set(key, value)
		}
	}

	url.RawQuery = query.Encode()
	patchedURL = url.String()
	return
}

// JoinURL updates the path part of the request url.
//     JoinURL("google.com", "admin") => "google.com/admin"
//     JoinURL("google.com?q=keyword", "admin") => "google.com/admin?q=keyword"
func JoinURL(originalURL string, paths ...interface{}) (joinedURL string, err error) {
	u, err := url.Parse(originalURL)
	if err != nil {
		return
	}

	var urlPaths = []string{u.Path}
	for _, p := range paths {
		urlPaths = append(urlPaths, fmt.Sprint(p))
	}

	if strings.HasSuffix(strings.Join(urlPaths, ""), "/") {
		u.Path = path.Join(urlPaths...) + "/"
	} else {
		u.Path = path.Join(urlPaths...)
	}

	joinedURL = u.String()
	return
}

// Stringify stringify any data, if it is a struct, will try to use its Name, Title, Code field, else will use its primary key
func Stringify(object interface{}) string {
	if obj, ok := object.(interface {
		Stringify() string
	}); ok {
		return obj.Stringify()
	}

	scope := gorm.Scope{Value: object}
	for _, column := range []string{"Name", "Title", "Code"} {
		if field, ok := scope.FieldByName(column); ok {
			if field.Field.IsValid() {
				result := field.Field.Interface()
				if valuer, ok := result.(driver.Valuer); ok {
					if result, err := valuer.Value(); err == nil {
						return fmt.Sprint(result)
					}
				}
				return fmt.Sprint(result)
			}
		}
	}

	if scope.PrimaryField() != nil {
		if scope.PrimaryKeyZero() {
			return ""
		}
		return fmt.Sprintf("%v#%v", scope.GetModelStruct().ModelType.Name(), scope.PrimaryKeyValue())
	}

	return fmt.Sprint(reflect.Indirect(reflect.ValueOf(object)).Interface())
}

// ModelType get value's model type
func ModelType(value interface{}) reflect.Type {
	reflectType := reflect.Indirect(reflect.ValueOf(value)).Type()

	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}

	return reflectType
}

// ParseTagOption parse tag options to hash
func ParseTagOption(str string) map[string]string {
	tags := strings.Split(str, ";")
	setting := map[string]string{}
	for _, value := range tags {
		v := strings.Split(value, ":")
		k := strings.TrimSpace(strings.ToUpper(v[0]))
		if len(v) == 2 {
			setting[k] = v[1]
		} else {
			setting[k] = k
		}
	}
	return setting
}

// ExitWithMsg debug error messages and print stack
func ExitWithMsg(msg interface{}, value ...interface{}) {
	fmt.Printf("\n"+filenameWithLineNum()+"\n"+fmt.Sprint(msg)+"\n", value...)
	debug.PrintStack()
}

// FileServer file server that disabled file listing
func FileServer(dir http.Dir) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Join(string(dir), r.URL.Path)
		if f, err := os.Stat(p); err == nil && !f.IsDir() {
			http.ServeFile(w, r, p)
			return
		}

		http.NotFound(w, r)
	})
}

func filenameWithLineNum() string {
	var total = 10
	var results []string
	for i := 2; i < 15; i++ {
		if _, file, line, ok := runtime.Caller(i); ok {
			total--
			results = append(results[:0],
				append(
					[]string{fmt.Sprintf("%v:%v", strings.TrimPrefix(file, os.Getenv("GOPATH")+"src/"), line)},
					results[0:]...)...)

			if total == 0 {
				return strings.Join(results, "\n")
			}
		}
	}
	return ""
}

// ParseTime parse time from string
var ParseTime = func(timeStr string) (time.Time, error) {
	return now.Parse(timeStr)
}

// FormatTime format time to string
var FormatTime = func(date time.Time, format string) string {
	return date.Format(format)
}

var replaceIdxRegexp = regexp.MustCompile(`\[\d+\]`)

// SortFormKeys sort form keys
func SortFormKeys(strs []string) {
	sort.Slice(strs, func(i, j int) bool { // true for first
		str1 := strs[i]
		str2 := strs[j]
		matched1 := replaceIdxRegexp.FindAllStringIndex(str1, -1)
		matched2 := replaceIdxRegexp.FindAllStringIndex(str2, -1)

		for x := 0; x < len(matched1); x++ {
			prefix1 := str1[:matched1[x][0]]
			prefix2 := str2

			if len(matched2) >= x+1 {
				prefix2 = str2[:matched2[x][0]]
			}

			if prefix1 != prefix2 {
				return strings.Compare(prefix1, prefix2) < 0
			}

			if len(matched2) < x+1 {
				return false
			}

			number1 := str1[matched1[x][0]:matched1[x][1]]
			number2 := str2[matched2[x][0]:matched2[x][1]]

			if number1 != number2 {
				if len(number1) != len(number2) {
					return len(number1) < len(number2)
				}
				return strings.Compare(number1, number2) < 0
			}
		}

		return strings.Compare(str1, str2) < 0
	})
}

// GetAbsURL get absolute URL from request, refer: https://stackoverflow.com/questions/6899069/why-are-request-url-host-and-scheme-blank-in-the-development-server
func GetAbsURL(req *http.Request) url.URL {
	var result url.URL

	if req.URL.IsAbs() {
		return *req.URL
	}

	if domain := req.Header.Get("Origin"); domain != "" {
		parseResult, _ := url.Parse(domain)
		result = *parseResult
	}

	result.Parse(req.RequestURI)
	return result
}

// Indirect returns last value that v points to
func Indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}
	return v
}

// SliceUniq removes duplicate values in given slice
func SliceUniq(s []string) []string {
	for i := 0; i < len(s); i++ {
		for i2 := i + 1; i2 < len(s); i2++ {
			if s[i] == s[i2] {
				// delete
				s = append(s[:i2], s[i2+1:]...)
				i2--
			}
		}
	}
	return s
}

// GetStructTagJSON is return tag
func GetStructTagJSON(f *gorm.Field) string {
	if string(f.Tag.Get("json")) == "" {
		return f.Name
	}
	return string(f.Tag.Get("json"))
}

// NewResponseMsg 创建求响应消息
func NewResponseMsg(msg interface{}) map[string]interface{} {
	return map[string]interface{}{"name": msg}
}

// Struct2Map 将结构体转为map
func Struct2Map(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}

// FillStruct 用map填充结构
func FillStruct(data map[string]interface{}, obj interface{}) error {
	for k, v := range data {
		err := SetField(obj, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetField 用map的值替换结构的值
func SetField(obj interface{}, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj).Elem()        //结构体属性值
	structFieldValue := structValue.FieldByName(name) //结构体单个属性值

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type() //结构体的类型
	val := reflect.ValueOf(value)              //map值的反射值

	var err error
	if structFieldType != val.Type() {
		val, err = TypeConversion(fmt.Sprintf("%v", value), structFieldValue.Type().Name()) //类型转换
		if err != nil {
			return err
		}
	}

	structFieldValue.Set(val)
	return nil
}

// TypeConversion 类型转换
func TypeConversion(value string, ntype string) (reflect.Value, error) {
	if ntype == "string" {
		return reflect.ValueOf(value), nil
	} else if ntype == "time.Time" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		return reflect.ValueOf(t), err
	} else if ntype == "Time" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		return reflect.ValueOf(t), err
	} else if ntype == "int" {
		i, err := strconv.Atoi(value)
		return reflect.ValueOf(i), err
	} else if ntype == "int8" {
		i, err := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(int8(i)), err
	} else if ntype == "int32" {
		i, err := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(int64(i)), err
	} else if ntype == "int64" {
		i, err := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(i), err
	} else if ntype == "float32" {
		i, err := strconv.ParseFloat(value, 64)
		return reflect.ValueOf(float32(i)), err
	} else if ntype == "float64" {
		i, err := strconv.ParseFloat(value, 64)
		return reflect.ValueOf(i), err
	}

	//else if .......增加其他一些类型的转换

	return reflect.ValueOf(value), errors.New("未知的类型：" + ntype)
}
