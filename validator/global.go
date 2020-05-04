package validator

type ConfigResp struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
	UserModelName string `json:"user_model_name"`
}

type DeleteReq struct {
	Ids string `json:"ids" comment:"标识符列表" validate:"required,min=1"`
}
