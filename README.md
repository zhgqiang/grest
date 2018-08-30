# Restful框架

## 查询
包含withCount, joins, groups, preloads, fields, where, order, offset, and limit字段。

|字段|字段类型|字段说明|
|-----|:---|:---|
|withCount|bool|是否返回总数|
|preloads|array|返回关联数据内容|
|fields|array|查询返回字段|
|where|array|查询条件|
|order|string|排序字段|
|offset|int|跳过数据|
|limit|int|查询数据长度|
|joins|array|关联表|
|groups|array|分组|