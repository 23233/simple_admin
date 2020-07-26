package simple_admin

type ConfigResp struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
	UserModelName string `json:"user_model_name"`
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
	Fields   []structInfo `json:"fields"`
	Autoincr string       `json:"autoincr"`
	Updated  string       `json:"updated"`
	Deleted  string       `json:"deleted"`
	Version  string       `json:"version"`
}

type CustomActionResp struct {
	Name    string       `json:"name"`    // action display name
	Methods string       `json:"methods"` // request run methods
	Fields  []structInfo `json:"fields"`  // request valid struct
	Path    string       `json:"path"`    // request path
}

// 搜索
type SearchReq struct {
	SearchText string `json:"search_text" form:"search_text" comment:"搜索内容" validate:"max=20" `
}
