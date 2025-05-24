package paypal

type PartnerReferral struct {
	IndividualOwners []struct {
		Names []struct {
			Prefix     string `json:"prefix"`
			GivenName  string `json:"given_name"`
			Surname    string `json:"surname"`
			MiddleName string `json:"middle_name"`
			Suffix     string `json:"suffix"`
			FullName   string `json:"full_name"`
			Type       string `json:"type"`
		} `json:"names,omitzero"`

		Addresses []struct {
			AddressLine1 string `json:"address_line_1"`
			AddressLine2 string `json:"address_line_2"`
			AdminArea2   string `json:"admin_area_2"`
			AdminArea1   string `json:"admin_area_1"`
			PostalCode   string `json:"postal_code"`
			CountryCode  string `json:"country_code"`
			Type         string `json:"type"`
		} `json:"addresses,omitzero"`

		Phones []struct {
			CountryCode     string `json:"country_code"`
			NationalNumber  string `json:"national_number"`
			ExtensionNumber string `json:"extension_number"`
			Type            string `json:"type"`
		} `json:"phones,omitzero"`

		Documents []struct {
			IdentificationNumber string `json:"identification_number"`
			IssuingCountryCode   string `json:"issuing_country_code"`
			Type                 string `json:"type"`
			Citizenship          string `json:"citizenship"`
			BirthDetails         struct {
				DateOfBirth string `json:"date_of_birth"`
			}
		}
	} `json:"individual_owners"`
}
