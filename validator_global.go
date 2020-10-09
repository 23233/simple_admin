package simple_admin

type ConfigResp struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
	UserModelName string `json:"user_model_name"`
}

type GetAllTableNameResp struct {
	Tables  map[string]string `json:"tables"`
	Remarks map[string]string `json:"remarks"`
}

type DeleteReq struct {
	Ids string `json:"ids" form:"ids" comment:"标识符列表" validate:"required,min=1"`
}

// 模型信息
type structInfo struct {
	Name         string `json:"name"`
	Types        string `json:"types"`
	MapName      string `json:"map_name"`
	XormTags     string `json:"xorm_tags"`
	SpTags       string `json:"sp_tags"`
	ValidateTags string `json:"validate_tags"`
	CommentTags  string `json:"comment_tags"`
	AttrTags     string `json:"attr_tags"`
}

type TableFieldsResp struct {
	Fields        []structInfo    `json:"fields"`
	AutoIncrement string          `json:"autoincr"`
	Updated       string          `json:"updated"`
	Deleted       string          `json:"deleted"`
	Created       map[string]bool `json:"created"`
	Version       string          `json:"version"`
}

type CustomActionResp struct {
	Name    string       `json:"name"`    // action display name
	Methods string       `json:"methods"` // request run methods
	Fields  []structInfo `json:"fields"`  // request valid struct
	Path    string       `json:"path"`    // request path
}

// 搜索
type SearchReq struct {
	Cols       []string `json:"cols" form:"cols" comment:"列信息" validate:"required"` // column map name
	SearchText string   `json:"search_text" form:"search_text" comment:"搜索内容" validate:"max=20" `
	FullMath   bool     `json:"full_math" form:"full_math" comment:"全匹配"`
}

// 获取数据源内容
type DashBoardGetDataItem struct {
	ColName     string `json:"col_name" comment:"列名" validate:"required,max=50"`
	OpType      string `json:"op_type" comment:"操作" validate:"required,max=20"` // = > != < >= <= in (not in) like
	Value       string `json:"value" comment:"值" validate:"required,max=100"`
	Order       uint64 `json:"order" comment:"顺序" validate:"required"` // 以小到大排列
	ConnectType string `json:"connect_type" comment:"连接方式"`            // and or xor not
}

// 获取数据源
type DashBoardGetDataReq struct {
	ColumnOp []DashBoardGetDataItem `json:"column_op"`
	Limit    uint64                 `json:"limit"`
}
