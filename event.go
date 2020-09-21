package simple_admin

// 插入之前
type SpInsertBeforeProcess interface {
	SpInsertBefore()
}

// 插入之后
type SpInsertAfterProcess interface {
	SpInsertAfter()
}

// 更新之前
type SpUpdateBeforeProcess interface {
	SpUpdateBefore()
}

// 更新之后
type SpUpdateAfterProcess interface {
	SpUpdateAfter()
}

// 删除之前
type SpDeleteBeforeProcess interface {
	SpDeleteBefore(uint64)
}

// 删除之后
type SpDeleteAfterProcess interface {
	SpDeleteAfter(uint64)
}

// 表别名
type SpTableNameProcess interface {
	Remark() string
}
