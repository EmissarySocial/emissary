package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func (domain Domain) GetBoolOK(name string) (bool, bool) {

	switch name {
	case "socialLinks":
		return domain.SocialLinks, true
	}

	return false, false
}

func (domain *Domain) GetObjectOK(name string) (any, bool) {

	switch name {
	case "signupForm":
		return &domain.SignupForm, true
	}

	return nil, false
}

func (domain Domain) GetStringOK(name string) (string, bool) {

	switch name {

	case "domainId":
		return domain.DomainID.Hex(), true

	case "label":
		return domain.Label, true

	case "headerHtml":
		return domain.HeaderHTML, true

	case "footerHtml":
		return domain.FooterHTML, true

	case "customCss":
		return domain.CustomCSS, true

	case "bannerUrl":
		return domain.BannerURL, true

	case "forward":
		return domain.Forward, true
	}

	return "", false
}

func (domain *Domain) SetBoolOK(name string, value bool) bool {

	switch name {

	case "socialLinks":
		domain.SocialLinks = value
		return true
	}

	return false
}

func (domain *Domain) SetStringOK(name string, value string) bool {

	switch name {

	case "domainId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			domain.DomainID = objectID
			return true
		}

	case "label":
		domain.Label = value
		return true

	case "headerHtml":
		domain.HeaderHTML = value
		return true

	case "footerHtml":
		domain.FooterHTML = value
		return true

	case "customCss":
		domain.CustomCSS = value
		return true

	case "bannerUrl":
		domain.BannerURL = value
		return true

	case "forward":
		domain.Forward = value
		return true
	}

	return false
}
