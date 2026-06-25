package resumegen

import "embed"

//go:embed defaultAppDir
var Defaults embed.FS

// ContainerfileRender is the Containerfile used to build the local render
// image (see container/render/Containerfile). It is embedded so the binary
// alone can build the image on demand without a source checkout.
//
//go:embed container/render/Containerfile
var ContainerfileRender []byte
