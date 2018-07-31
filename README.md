# Restful框架

## 查询
包含withCount, includes, fields, where, order, offset, and limit字段。fields可单独使用，table、join、
fields需联合使用,如果以此方式使用，返回值为fields里字段。

|字段|字段类型|
|-----|:---|
|withCount|bool|
|include|string或array|
|fields|array|
|where|array|
|order|string|
|offset|int|
|limit|int|