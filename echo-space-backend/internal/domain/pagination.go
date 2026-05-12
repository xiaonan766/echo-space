package domain

type PaginationResult[T any] struct {
	TotalCount int64 `json:"totalCount"`
	PageSize   int   `json:"pageSize"`
	PageNo     int   `json:"pageNo"`
	PageTotal  int   `json:"pageTotal"`
	List       []T   `json:"list"`
}

func NewPaginationResult[T any](list []T, totalCount int64, pageNo int, pageSize int) PaginationResult[T] {
	if list == nil {
		list = []T{}
	}

	pageTotal := 0
	if pageSize > 0 {
		pageTotal = int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	}

	return PaginationResult[T]{
		TotalCount: totalCount,
		PageSize:   pageSize,
		PageNo:     pageNo,
		PageTotal:  pageTotal,
		List:       list,
	}
}
