module github.com/coze-dev/cozeloop-go

go 1.18

require (
	github.com/bluele/gcache v0.0.2
	github.com/bytedance/mockey v1.2.14
	github.com/coze-dev/cozeloop-go/spec v0.1.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/smartystreets/goconvey v1.8.1
	github.com/valyala/fasttemplate v1.2.2
	golang.org/x/sync v0.11.0
)

require (
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/arch v0.11.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
)

replace (
	github.com/coze-dev/cozeloop-go/spec => ./spec
)