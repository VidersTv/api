package modelstructures

import "github.com/viderstv/common/structures"

type Color structures.Color

func (c Color) ToModel() int {
	return int(c)
}
