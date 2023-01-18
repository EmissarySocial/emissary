package model

/*********************************
 * Getter Methods
 *********************************/

func (content *Content) GetStringOK(name string) (string, bool) {
	switch name {

	case "format":
		return content.Format, true

	case "raw":
		return content.Raw, true

	case "html":
		return content.HTML, true

	default:
		return "", false
	}
}

/*********************************
 * Setter Methods
 *********************************/

func (content *Content) SetStringOK(name string, value string) bool {
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
