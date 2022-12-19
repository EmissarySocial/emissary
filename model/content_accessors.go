package model

/*********************************
 * Getter Methods
 *********************************/

func (content *Content) GetString(name string) string {
	switch name {
	case "format":
		return content.Format
	case "raw":
		return content.Raw
	case "html":
		return content.HTML
	default:
		return ""
	}
}

/*********************************
 * Setter Methods
 *********************************/

func (content *Content) SetString(name string, value string) bool {
	switch name {
	case "format":
		content.Format = value
		return true
	case "raw":
		content.Raw = value
		return true
	case "html":
		content.HTML = value
		return true
	default:
		return false
	}
}
