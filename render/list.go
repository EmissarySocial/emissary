package render

type List []Renderer

func (rs List) First() Renderer {
	return rs[0]
}

func (rs List) Last() Renderer {
	if len(rs) == 0 {
		return nil
	}

	return rs[len(rs)-1]
}

func (rs List) IsLength(length int) bool {
	return length == len(rs)
}

func (rs List) IsEmpty() bool {
	return len(rs) == 0
}
