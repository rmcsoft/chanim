package chanim

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"syscall"
)

// PackedPixmap packed pixmap
type PackedPixmap struct {
	Data      []byte
	Width     int
	Height    int
	PixFormat PixelFormat
}

// Save saves PackedPixmap
func (pp *PackedPixmap) Save(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	header := []uint32{
		uint32(pp.PixFormat),
		uint32(pp.Width),
		uint32(pp.Height),
	}
	for _, v := range header {
		err = binary.Write(file, binary.LittleEndian, v)
		if err != nil {
			return err
		}
	}

	_, err = file.Write(pp.Data)
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
func (pp *PackedPixmap) Unpack() (*Pixmap, error) {
	pixSize := GetPixelSize(pp.PixFormat)

	unpackedDataSize := pp.Width * pp.Height * pixSize
	unpackedData := make([]byte, 0, unpackedDataSize)

	pix := make([]byte, pixSize)

	rowCount := 0
	rowSize := 0
	for pos := 0; pos < len(pp.Data); {
		pixCount := int(pp.Data[pos])
		if pixCount == 0 {
			// New row
			if rowSize != pp.Width {
				return nil, errors.New("Invalid data")
			}

			rowCount++
			rowSize = 0

			pos++
			continue
		}
		pos++
		if pos+pixSize >= len(pp.Data) {
			return nil, errors.New("Invalid data")
		}
		copy(pix, pp.Data[pos:pos+pixSize])
		for i := 0; i < pixCount; i++ {
			unpackedData = append(unpackedData, pix...)
		}

		rowSize += pixCount
		pos += pixSize
	}

	if rowCount != pp.Height {
		return nil, errors.New("Invalid data")
	}

	pixmap := &Pixmap{
		Data:        unpackedData,
		Width:       pp.Width,
		Height:      pp.Height,
		PixFormat:   pp.PixFormat,
		BytePerLine: pp.Width * pixSize,
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

const rawHeaderSize = 3 * 4

type rawHeader [3]uint32

func parseHeader(reader io.Reader) (*PackedPixmap, error) {
	header := rawHeader{}
	for i := 0; i < len(header); i++ {
		err := binary.Read(reader, binary.LittleEndian, &header[i])
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
	return &PackedPixmap{
		Width:     width,
		Height:    height,
		PixFormat: pixFormat,
	}, nil
}

// Check checks PackedPixmap
func (pp *PackedPixmap) Check() error {
	if pp.Width*pp.Height > 0 && len(pp.Data) == 0 {
		return errors.New("Invalid data")
	}

	pixSize := GetPixelSize(pp.PixFormat)
	rowCount := 0
	rowSize := 0
	for pos := 0; pos < len(pp.Data); {
		pixCount := pp.Data[pos]
		if pixCount == 0 {
			// New row
			if rowSize != pp.Width {
				return errors.New("Invalid data")
			}

			rowCount++
			rowSize = 0

			pos++
			continue
		}

		rowSize += int(pixCount)
		pos += 1 + pixSize
	}

	if rowCount != pp.Height {
		return errors.New("Invalid data")
	}

	return nil
}

// LoadPackedPixmap loads LoadPacked
func LoadPackedPixmap(fileName string) (*PackedPixmap, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	pp, err := parseHeader(file)
	if err != nil {
		return nil, err
	}

	pp.Data, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = pp.Check()
	if err != nil {
		return nil, err
	}

	return pp, nil
}

// MMapPackedPixmap maps PackedPixmap to memory
func MMapPackedPixmap(fileName string) (*PackedPixmap, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := int(fileInfo.Size())

	data, err := syscall.Mmap(int(file.Fd()), 0, fileSize, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}

	pp, err := parseHeader(bytes.NewReader(data[0:rawHeaderSize]))
	if err != nil {
		return nil, err
	}
	pp.Data = data[rawHeaderSize:]

	err = pp.Check()
	if err != nil {
		return nil, err
	}

	return pp, nil
}

func eqPixels(a []byte, b []byte) bool {
	return bytes.Equal(a, b)
}

// PackPixmap packs Pixmap
func PackPixmap(pixmap *Pixmap) (*PackedPixmap, error) {
	pp := &PackedPixmap{
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

			pp.Data = append(pp.Data, eqPixCount)
			pp.Data = append(pp.Data, packedPixel...)
		}
		pp.Data = append(pp.Data, 0x00) // New row
	}

	return pp, nil
}
