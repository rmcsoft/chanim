package chanim

import (
	"errors"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

// Pixmap contains a collection of pixels
type Pixmap struct {
	Data        []byte
	Width       int
	Height      int
	BytePerLine int
	PixFormat   PixelFormat
}

type pixmapKey struct {
	fileName  string
	pixFormat PixelFormat
}

func init() {
	img.Init(img.INIT_JPG | img.INIT_PNG)
}

func pixelFormatToSDL(pixelFormat PixelFormat) (uint32, error) {
	switch pixelFormat {
	case RGB16:
		return sdl.PIXELFORMAT_RGB565, nil
	case RGB32:
		return sdl.PIXELFORMAT_ARGB8888, nil
	default:
		return 0, errors.New("Unsupported pixel format")
	}
}

// LoadPixmap loads Pixmap from file
func LoadPixmap(fileName string, pixFormat PixelFormat) (*Pixmap, error) {
	image, err := img.Load(fileName)
	if err != nil {
		return nil, err
	}
	defer image.Free()

	sdlPixFormat, err := pixelFormatToSDL(pixFormat)
	if err != nil {
		return nil, err
	}

	convertedImage, err := image.ConvertFormat(sdlPixFormat, 0)
	if err != nil {
		return nil, err
	}
	defer convertedImage.Free()

	pixmap := Pixmap{
		Data:        make([]byte, len(convertedImage.Pixels())),
		Width:       int(convertedImage.W),
		Height:      int(convertedImage.H),
		BytePerLine: int(convertedImage.Pitch),
		PixFormat:   pixFormat,
	}
	copy(pixmap.Data, convertedImage.Pixels())
	return &pixmap, nil
}
