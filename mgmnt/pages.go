package mgmnt

// pageSize is how many items each list request asks for.
const pageSize = 50

// listPages fetches a list page by page, starting at offset, until it has
// limit items (every item when limit <= 0) or the API has no more. fetch
// returns one page and the API's total count (meta.count, 0 when the API
// doesn't send one). listPages returns the items and the total, or -1 when
// the API didn't report one.
//
// When the API reports a total, paging stops at it, so a page shorter than
// asked for (an API that caps the page size) doesn't end the list early.
func listPages[T any](limit, offset int, fetch func(limit, offset int) ([]T, int, error)) ([]T, int, error) {
	items := []T{}
	total := -1
	for limit <= 0 || len(items) < limit {
		n := pageSize
		if limit > 0 && limit-len(items) < n {
			n = limit - len(items)
		}
		page, count, err := fetch(n, offset)
		if err != nil {
			return nil, total, err
		}
		if count > 0 {
			total = count
		}
		items = append(items, page...)
		offset += len(page)
		if len(page) == 0 || (total >= 0 && offset >= total) || (total < 0 && len(page) < n) {
			break
		}
	}
	return items, total, nil
}

// listTotal is the total to report for a list: the API's count when it sent
// one, else how far the list got.
func listTotal(total, offset, fetched int) int {
	if total >= 0 {
		return total
	}
	return offset + fetched
}
