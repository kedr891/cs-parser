package swagger

import _ "embed"

//go:embed skins_api/skins.swagger.json
var skinsSpec []byte

func Skins() []byte {
	return skinsSpec
}
