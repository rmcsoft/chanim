package chanim

// PixelFormat is an enumeration of pixel formats
type PixelFormat int

const (
	// RGB32 is 32-bit RGB format (0xffRRGGBB)
	RGB32 PixelFormat = iota
	// RGB16 is 16-bit RGB format (5-6-5)
	RGB16
)

// GetPixelSize gets pixel size
func GetPixelSize(pixFormat PixelFormat) int {
	switch pixFormat {
	case RGB32:
		return 4
	case RGB16:
		return 2
	default:
		panic("Unsupported PixelFormat")
	}
}

func GetBPP(pixFormat PixelFormat) int {
	switch pixFormat {
	case RGB32:
		return 32
	case RGB16:
		return 16
	default:
		panic("Unsupported PixelFormat")
	}
}
