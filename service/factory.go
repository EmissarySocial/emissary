package service

type Factory interface {
	Circle() *Circle
	Domain() *Domain
	Folder() *Folder
	Group() *Group
	MerchantAccount() *MerchantAccount
	Product() *Product
	Registration() *Registration
	SearchTag() *SearchTag
	Stream() *Stream
	Template() *Template
	Theme() *Theme
}
