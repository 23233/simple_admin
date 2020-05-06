package simple_admin

type ConfigResp struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
	UserModelName string `json:"user_model_name"`
}

type DeleteReq struct {
	Ids string `json:"ids" comment:"标识符列表" validate:"required,min=1"`
}

type TableFieldsResp struct {
	Fields   []structInfo `json:"fields"`
	Autoincr string       `json:"autoincr"`
	Updated  string       `json:"updated"`
	Deleted  string       `json:"deleted"`
	Version  string       `json:"version"`
}
