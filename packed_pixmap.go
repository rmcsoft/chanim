package chanim

import (
	"encoding/binary"
	"errors"
	"io/ioutil"
	"os"
)

// PackedPixmap packed pixmap
type PackedPixmap struct {
	Data      []byte
	Width     int
	Height    int
	PixFormat PixelFormat
}

// Save saves PackedPixmap
func (packedPixmap *PackedPixmap) Save(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	header := []uint32{
		uint32(packedPixmap.PixFormat),
		uint32(packedPixmap.Width),
		uint32(packedPixmap.Height),
	}
	for _, v := range header {
		err = binary.Write(file, binary.LittleEndian, v)
		if err != nil {
			return err
		}
	}

	_, err = file.Write(packedPixmap.Data)
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

// Unpack unpacks PackedPixmap
func (packedPixmap *PackedPixmap) Unpack() (*Pixmap, error) {
	pixSize := GetPixelSize(packedPixmap.PixFormat)

	unpackedDataSize := packedPixmap.Width * packedPixmap.Height * pixSize
	unpackedData := make([]byte, 0, unpackedDataSize)

	pix := make([]byte, pixSize)

	rowCount := 0
	rowSize := 0
	for pos := 0; pos < len(packedPixmap.Data); {
		pixCount := int(packedPixmap.Data[pos])
		if pixCount == 0 {
			// New row
			if rowSize != packedPixmap.Width {
				return nil, errors.New("Invalid data")
			}

			rowCount++
			rowSize = 0

			pos++
			continue
		}
		pos++
		if pos+pixSize >= len(packedPixmap.Data) {
			return nil, errors.New("Invalid data")
		}
		copy(pix, packedPixmap.Data[pos:pos+pixSize])
		for i := 0; i < pixCount; i++ {
			unpackedData = append(unpackedData, pix...)
		}

		rowSize += pixCount
		pos += pixSize
	}

	if rowCount != packedPixmap.Height {
		return nil, errors.New("Invalid data")
	}

	pixmap := &Pixmap{
		Data:        unpackedData,
		Width:       packedPixmap.Width,
		Height:      packedPixmap.Height,
		PixFormat:   packedPixmap.PixFormat,
		BytePerLine: packedPixmap.Width * pixSize,
	}
	return pixmap, nil
}

func u32ToPixFormat(val uint32) (PixelFormat, error) {
	switch val {
	case uint32(RGB16):
		return RGB16, nil
	case uint32(RGB32):
		return RGB32, nil
	default:
		return 0, errors.New("Unsupported PixelFormat")
	}
}

// LoadPackedPixmap loads LoadPacked
func LoadPackedPixmap(fileName string) (*PackedPixmap, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	header := [3]uint32{}
	for i := 0; i < len(header); i++ {
		err = binary.Read(file, binary.LittleEndian, &header[i])
		if err != nil {
			return nil, err
		}
	}
	pixFormat, err := u32ToPixFormat(header[0])
	if err != nil {
		return nil, err
	}
	width := int(header[1])
	if width < 0 || width > 32000 {
		return nil, errors.New("Invalid width")
	}
	height := int(header[2])
	if height < 0 || height > 32000 {
		return nil, errors.New("Invalid height")
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if width*height > 0 && len(data) == 0 {
		return nil, errors.New("Invalid data")
	}

	pixSize := GetPixelSize(pixFormat)
	rowCount := 0
	rowSize := 0
	for pos := 0; pos < len(data); {
		pixCount := data[pos]
		if pixCount == 0 {
			// New row
			if rowSize != width {
				return nil, errors.New("Invalid data")
			}

			rowCount++
			rowSize = 0

			pos++
			continue
		}

		rowSize += int(pixCount)
		pos += 1 + pixSize
	}

	if rowCount != height {
		return nil, errors.New("Invalid data")
	}
	packedPixmap := &PackedPixmap{
		Data:      data,
		Width:     width,
		Height:    height,
		PixFormat: pixFormat,
	}
	return packedPixmap, nil
}

func eqPixels(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// PackPixmap packs Pixmap
func PackPixmap(pixmap *Pixmap) (*PackedPixmap, error) {
	packedPixmap := &PackedPixmap{
		Width:     pixmap.Width,
		Height:    pixmap.Height,
		PixFormat: pixmap.PixFormat,
	}

	pixSize := GetPixelSize(pixmap.PixFormat)
	packedPixel := make([]byte, pixSize)

	for y := 0; y < pixmap.Height; y++ {
		rowOffset := pixmap.BytePerLine * y
		row := pixmap.Data[rowOffset : rowOffset+pixmap.Width*pixSize]

		for pixOffset := 0; pixOffset <= len(row)-pixSize; {
			copy(packedPixel, row[pixOffset:pixOffset+pixSize])

			var eqPixCount byte = 1
			pixOffset += pixSize
			for pixOffset <= len(row)-pixSize && eqPixCount < 0xFF {
				if !eqPixels(packedPixel, row[pixOffset:pixOffset+pixSize]) {
					break
				}

				eqPixCount++
				pixOffset += pixSize
			}

			packedPixmap.Data = append(packedPixmap.Data, eqPixCount)
			packedPixmap.Data = append(packedPixmap.Data, packedPixel...)
		}
		packedPixmap.Data = append(packedPixmap.Data, 0x00) // New row
	}

	return packedPixmap, nil
}
