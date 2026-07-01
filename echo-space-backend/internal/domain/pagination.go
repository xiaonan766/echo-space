package domain

type PaginationResult[T any] struct {
	TotalCount int64 `json:"totalCount"`
	PageSize   int   `json:"pageSize"`
	PageNo     int   `json:"pageNo"`
	PageTotal  int   `json:"pageTotal"`
	List       []T   `json:"list"`
}

type CursorPaginationResult[T any] struct {
	List       []T    `json:"list"`
	PageSize   int    `json:"pageSize"`
	NextCursor string `json:"nextCursor"`
	PrevCursor string `json:"prevCursor"`
	HasNext    bool   `json:"hasNext"`
	HasPrev    bool   `json:"hasPrev"`
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

func NewCursorPaginationResult[T any](list []T, pageSize int, nextCursor string, prevCursor string, hasNext bool, hasPrev bool) CursorPaginationResult[T] {
	if list == nil {
		list = []T{}
	}

	return CursorPaginationResult[T]{
		List:       list,
		PageSize:   pageSize,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}
}
