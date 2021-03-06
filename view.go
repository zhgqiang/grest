package grest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
)

// APIView Model construct
type APIView struct{}

// toPrimaryQueryParams generate query params based on primary key, multiple primary value are linked with a comma
func (p *APIView) toPrimaryQueryParams(result interface{}, primaryValue string, context *Context) (string, []interface{}) {
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

// findCount2 query data count
// NOTE: not use
func (p *APIView) findCount2(result interface{}, filter map[string]interface{}, context *Context) (int, error) {
	db := context.GetDB()
	if db == nil {
		return 0, errors.New("db is nil")
	}
	db = db.Begin()
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

// findCount query data count
func (p *APIView) findCount(result interface{}, where []interface{}, context *Context) (int, error) {
	db := context.GetDB()
	if db == nil {
		return 0, errors.New("db is nil")
	}
	db = db.Begin()
	db = db.Find(result)
	if where != nil {
		if len(where) == 1 {
			db = db.Where(where[0])
		} else if len(where) > 1 {
			db = db.Where(where[0], where[1:]...)
		}
	}
	count := 0
	if db.Error != nil {
		return 0, db.Error
	}
	db = db.Count(&count)
	if db == nil {
		return 0, errors.New("db is nil")
	}
	if db.Error != nil {
		return 0, db.Error
	}
	if db.Commit(); db.Error != nil {
		return 0, db.Error
	}
	return count, nil
}

// FindMany2 query data
// NOTE: not use
func (p *APIView) FindMany2(result interface{}, filter map[string]interface{}, context *Context) (int, error) {
	db := context.GetDB()
	if db == nil {
		return 0, errors.New("db is nil")
	}
	db = db.Begin()
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
					c, err := p.findCount2(result, filter, context)
					if err != nil {
						return 0, err
					}
					count = c
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

// FindMany query data
func (p *APIView) FindMany(result interface{}, filter *Filter, context *Context) (int, error) {
	db := context.GetDB()
	if db == nil {
		return 0, errors.New("db is nil")
	}
	db = db.Begin()
	var count = 0
	if filter != nil {
		// query fields
		if filter.Fields != nil && len(filter.Fields) > 0 {
			db = db.Select(filter.Fields)
		}

		// query result sorting
		if filter.Order != "" {
			db = db.Order(filter.Order)
		}

		// query by where condition
		if filter.Where != nil && len(filter.Where) > 0 {
			if len(filter.Where) == 1 {
				db = db.Where(filter.Where[0])
			} else if len(filter.Where) > 1 {
				db = db.Where(filter.Where[0], filter.Where[1:]...)
			}
		}

		// whether the query count
		c, err := p.findCount(result, filter.Where, context)
		if err != nil {
			return 0, err
		}
		count = c

		if filter.Joins != nil && len(filter.Joins) > 0 {
			for _, join := range filter.Joins {
				db = db.Joins(join)
			}
		}

		if filter.Groups != nil && len(filter.Groups) > 0 {
			for _, group := range filter.Groups {
				db = db.Group(group)
			}
		}

		// query contains related data
		if filter.Preloads != nil && len(filter.Groups) > 0 {
			if len(filter.Groups) == 1 {
				db = db.Preload(filter.Groups[0])
			} else if len(filter.Groups) > 1 {
				db = db.Preload(filter.Groups[0], filter.Groups[1:])
			}
		}

		// query offset
		// query limit
		// if filter.Offset != "" && filter.Limit != "" {
		// 	offset, err := strconv.Atoi(filter.Offset)
		// 	if err != nil {
		// 		return 0, fmt.Errorf("offset format is incorrect,%v", err.Error())
		// 	}
		// 	limit, err := strconv.Atoi(filter.Limit)
		// 	if err != nil {
		// 		return 0, fmt.Errorf("limit format is incorrect,%v", err.Error())
		// 	}
		// 	db = db.Limit(limit).Offset(offset)
		// }

		if filter.Limit != 0 {
			db = db.Limit(filter.Limit)
			db = db.Offset(filter.Offset)
		}
	}
	db = db.Find(result)
	if db == nil {
		return 0, errors.New("db is nil")
	}
	if db.Error != nil {
		return 0, db.Error
	}
	if db.Commit(); db.Error != nil {
		return 0, db.Error
	}
	return count, nil
}

// Save is Model create
func (p *APIView) Save(result interface{}, context *Context) error {
	db := context.GetDB()
	if db == nil {
		return errors.New("db is nil")
	}
	if db.NewScope(result).PrimaryKeyZero() {
		return db.Create(result).Error
	}
	return db.Save(result).Error
}

// FindOne Model query one data
func (p *APIView) FindOne(result interface{}, context *Context) error {
	primaryQuerySQL, primaryParams := p.toPrimaryQueryParams(result, context.ResourceID, context)
	db := context.GetDB()
	if db == nil {
		return errors.New("db is nil")
	}
	if primaryQuerySQL != "" {
		return db.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error
	}

	return errors.New("failed to find")
}

// Delete Model delete one data
func (p *APIView) Delete(result interface{}, context *Context) error {
	db := context.GetDB()
	if db == nil {
		return errors.New("db is nil")
	}
	if !db.Find(result).RecordNotFound() {
		return db.Delete(result).Error
	}
	return gorm.ErrRecordNotFound
}
