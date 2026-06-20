// Separate module: dev/test-only. Its dependencies (a mature reference EXIF
// reader) never reach the exifscalpel library or its consumers' builds.
// See ../CONTRIBUTING.md "Dependency policy".
module codeberg.org/elkarrde/exifscalpel/conformance

go 1.22

require (
	codeberg.org/elkarrde/exifscalpel v0.0.0
	github.com/dsoprea/go-exif/v3 v3.0.1
)

require (
	github.com/dsoprea/go-logging v0.0.0-20200710184922-b02d349568dd // indirect
	github.com/dsoprea/go-utility/v2 v2.0.0-20221003172846-a3e1774ef349 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551 // indirect
	golang.org/x/net v0.0.0-20221002022538-bcab6841153b // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace codeberg.org/elkarrde/exifscalpel => ../
