module github.com/benpate/ghost

go 1.17

replace (
	github.com/benpate/activitystream => ../activitystream
	github.com/benpate/compare => ../compare
	github.com/benpate/nebula => ../nebula
	github.com/benpate/convert => ../convert
	github.com/benpate/data => ../data
	github.com/benpate/data-mock => ../data-mock
	github.com/benpate/data-mongo => ../data-mongo
	github.com/benpate/datatype => ../datatype
	github.com/benpate/derp => ../derp
	github.com/benpate/exp => ../exp
	github.com/benpate/first => ../first
	github.com/benpate/form => ../form
	github.com/benpate/html => ../html
	github.com/benpate/htmlconv => ../htmlconv
	github.com/benpate/id => ../id
	github.com/benpate/list => ../list
	github.com/benpate/mediaserver => ../mediaserver
	github.com/benpate/nebula => ../nebula
	github.com/benpate/path => ../path
	github.com/benpate/schema => ../schema
	github.com/benpate/steranko => ../steranko
)

require (
	github.com/benpate/activitystream v0.2.1
	github.com/benpate/compare v0.0.4
	github.com/benpate/nebula v0.0.1
	github.com/benpate/convert v0.11.0
	github.com/benpate/data v0.11.1
	github.com/benpate/data-mongo v0.0.2
	github.com/benpate/datatype v0.0.3
	github.com/benpate/derp v0.22.0
	github.com/benpate/exp v0.0.2
	github.com/benpate/first v0.0.2
	github.com/benpate/form v0.3.9
	github.com/benpate/html v0.11.3
	github.com/benpate/id v0.0.0-20211223035835-658efd027b24
	github.com/benpate/list v0.4.1
	github.com/benpate/mediaserver v0.0.0
	github.com/benpate/null v0.6.0
	github.com/benpate/path v0.4.1
	github.com/benpate/schema v0.10.2
	github.com/benpate/steranko v0.4.3
	github.com/davecgh/go-spew v1.1.1
	github.com/fsnotify/fsnotify v1.5.1
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/gorilla/feeds v1.1.1
	github.com/labstack/echo/v4 v4.6.1
	github.com/mmcdole/gofeed v1.1.3
	github.com/spf13/afero v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tdewolff/minify/v2 v2.9.24
	go.mongodb.org/mongo-driver v1.8.1
)

require (
	cloud.google.com/go v0.99.0 // indirect
	cloud.google.com/go/storage v1.18.2 // indirect
	github.com/PuerkitoBio/goquery v1.8.0 // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/benpate/exiffix v0.1.0 // indirect
	github.com/benpate/htmlconv v0.3.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/chai2010/webp v1.1.0 // indirect
	github.com/cncf/udpa/go v0.0.0-20210930031921-04548b0d99d4 // indirect
	github.com/cncf/xds/go v0.0.0-20211011173535-cb28da3451f1 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/envoyproxy/go-control-plane v0.9.10-0.20210907150352-cf90f659a021 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/googleapis/google-cloud-go-testing v0.0.0-20210719221736-1c9a4c676720 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mmcdole/goxpp v0.0.0-20200921145534-2f3784f67354 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/muesli/smartcrop v0.3.0 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd // indirect
	github.com/tdewolff/parse/v2 v2.5.26 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3 // indirect
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.63.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211223182754-3ac035c7e7cb // indirect
	google.golang.org/grpc v1.43.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
