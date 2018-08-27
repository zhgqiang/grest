package grest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
)

type APIView struct{}

// ToPrimaryQueryParams generate query params based on primary key, multiple primary value are linked with a comma
func (this *APIView) toPrimaryQueryParams(result interface{}, primaryValue string, context *Context) (string, []interface{}) {
	if primaryValue != "" {
		scope := context.GetDB().NewScope(result)

		if primaryField := scope.PrimaryField(); primaryField != nil {
			primaryFields := []*gorm.StructField{primaryField.StructField}

			if len(primaryFields) > 1 {
				if primaryValueStrs := strings.Split(primaryValue, ","); len(primaryValueStrs) == len(primaryFields) {
					sqls := make([]string, 0)
					primaryValues := make([]interface{}, 0)
					for idx, field := range primaryFields {
						sqls = append(sqls, fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(field.DBName)))
						primaryValues = append(primaryValues, primaryValueStrs[idx])
					}

					return strings.Join(sqls, " AND "), primaryValues
				}
			}

			// fallback to first configured primary field
			if len(primaryFields) > 0 {
				return fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(primaryFields[0].DBName)), []interface{}{primaryValue}
			}

		}
		//multiple primary fields

		// if no configured primary fields found
		if primaryField := scope.PrimaryField(); primaryField != nil {
			return fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(primaryField.DBName)), []interface{}{primaryValue}
		}
	}

	return "", []interface{}{}
}

func (this *APIView) findCount(result interface{}, filter map[string]interface{}, context *Context) (int, error) {
	db := context.GetDB().Begin()
	db = db.Find(result)
	if filter != nil {
		if where, ok := filter["where"]; ok {
			if whereArr, ok := where.([]interface{}); ok {
				if len(whereArr) == 1 {
					db = db.Where(whereArr[0])
				} else if len(whereArr) > 1 {
					db = db.Where(whereArr[0], whereArr[1:]...)
				}
			} else {
				return 0, errors.New("where format is incorrect, non-array")
			}
		}
	}
	count := 0
	if db.Count(&count).Commit(); db.Error != nil {
		return 0, db.Error
	}
	return count, nil
}

func (this *APIView) FindMany(result interface{}, filter map[string]interface{}, context *Context) (int, error) {
	db := context.GetDB().Begin()
	var count = 0
	if filter != nil {
		// query fields
		if fields, ok := filter["fields"]; ok {
			if fieldsArr, ok := fields.([]interface{}); ok {
				db = db.Select(fieldsArr)
			} else {
				return 0, errors.New("fields format is incorrect, non-array")
			}
		}

		// query result sorting
		if order, ok := filter["order"]; ok {
			db = db.Order(order)
		}

		// query by where condition
		if where, ok := filter["where"]; ok {
			if whereArr, ok := where.([]interface{}); ok {
				if len(whereArr) == 1 {
					db = db.Where(whereArr[0])
				} else if len(whereArr) > 1 {
					db = db.Where(whereArr[0], whereArr[1:]...)
				}
			} else {
				return 0, errors.New("where format is incorrect, non-array")
			}
		}

		// whether the query count
		if withCount, ok := filter["withCount"]; ok {
			if withCountBool, ok := withCount.(bool); ok {
				if withCountBool {
					if c, err := this.findCount(result, filter, context); err != nil {
						return 0, err
					} else {
						count = c
					}
				}
			} else {
				return 0, errors.New("withCount format is incorrect, non-bool")
			}
		}

		joins, joinsOk := filter["joins"]
		groups, groupOk := filter["groups"]

		if joinsOk && groupOk {
			joinArr, joinArrOk := joins.([]interface{})
			groupArr, groupArrOk := groups.([]interface{})
			if joinArrOk && groupArrOk {
				for _, join := range joinArr {
					if joinStr, ok := join.(string); ok {
						db = db.Joins(joinStr)
					}
				}
				for _, group := range groupArr {
					if groupStr, ok := group.(string); ok {
						db = db.Group(groupStr)
					}
				}
			}
		}

		// query contains related data
		if includes, ok := filter["include"]; ok {
			switch includes.(type) {
			case string:
				db = db.Preload(includes.(string))
			case []interface{}:
				for _, include := range includes.([]interface{}) {
					switch include.(type) {
					case string:
						db = db.Preload(include.(string))
					case []interface{}:
						arr := include.([]interface{})
						if len(arr) == 1 {
							if a, ok := arr[0].(string); ok {
								db = db.Preload(a)
							} else {
								return 0, errors.New("include format is incorrect, the first element of the array is not a string")
							}
						} else if len(arr) > 1 {
							if a, ok := arr[0].(string); ok {
								db = db.Preload(a, arr[1:])
							} else {
								return 0, errors.New("include format is incorrect, the first element of the array is not a string")
							}
						}
					default:
						return 0, errors.New("include format is incorrect,non-string or non-array")
					}
				}
			default:
				return 0, errors.New("include format is incorrect,non-string or non-array")
			}
		}

		// query offset
		offset, offsetOk := filter["offset"]

		// query limit
		limit, limitOk := filter["limit"]

		// exist offset and limit
		if offsetOk && limitOk {
			db = db.Limit(limit).Offset(offset)
		}
	}
	if db.Find(result).Commit(); db.Error != nil {
		return 0, db.Error
	}
	return count, nil
}

func (this *APIView) Save(result interface{}, context *Context) error {
	if context.GetDB().NewScope(result).PrimaryKeyZero() {
		return context.GetDB().Create(result).Error
	} else {
		return context.GetDB().Save(result).Error
	}
}

func (this *APIView) FindOne(result interface{}, context *Context) error {
	primaryQuerySQL, primaryParams := this.toPrimaryQueryParams(result, context.ResourceID, context)

	if primaryQuerySQL != "" {
		return context.GetDB().First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error
	}

	return errors.New("failed to find")
}

func (this *APIView) Delete(result interface{}, context *Context) error {
	if !context.GetDB().Find(result).RecordNotFound() {
		return context.GetDB().Delete(result).Error
	}
	return gorm.ErrRecordNotFound
}
