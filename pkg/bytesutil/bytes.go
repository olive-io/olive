package bytesutil

// PathJoin joins any number of path elements into a single path,
// separating them with slashes. Empty elements are ignored.
// However, if the argument list is empty or all its elements
// are empty, Join returns an empty []byte.
func PathJoin(elem ...[]byte) []byte {
	size := 0
	for _, e := range elem {
		size += len(e)
	}
	if size == 0 {
		return []byte("")
	}
	buf := make([]byte, 0, size+len(elem)-1)
	for _, e := range elem {
		if len(buf) > 0 || e != nil {
			if len(buf) > 0 {
				buf = append(buf, '/')
			}
			buf = append(buf, e...)
		}
	}

	return buf
}
