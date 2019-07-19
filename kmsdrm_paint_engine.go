package chanim

/*
#cgo CFLAGS: -O3

#include <stdbool.h>
#include <stddef.h>
#include <string.h>
#include <stdlib.h>
#include <stdint.h>

typedef struct {
	int x;
	int y;
 	int width;
 	int height;
} Rect;

typedef struct {
	Rect rect;
	char* data;
	int dataSize;
	int bytePerLine;
} Pixmap;

typedef enum {
	ccClearRect,
	ccDrawPixmap,
	ccDrawPackedPixmap
} CmdCode;

typedef struct {
	CmdCode code;
	union {
		Pixmap pixmap;
		Rect rect;
	} data;
} Cmd;

static inline
int min(int a, int b) {
	return a < b ? a : b;
}

static inline
int max(int a, int b) {
	return a > b ? a : b;
}

static
Rect intersect(const Rect* r1, const Rect* r2) {
	Rect ret;

	ret.x = max(r1->x, r2->x);
	ret.width = min(r1->x + r1->width, r2->x + r2->width) - ret.x;
	if (ret.width < 0)
		ret.width = 0;

	ret.y = max(r1->y, r2->y);
	ret.height = min(r1->y + r1->height, r2->y + r2->height) - ret.y;
	if (ret.height < 0)
		ret.height = 0;

	return ret;
}

static inline
bool isRectNull(const Rect* r) {
	return r->width == 0 && r->height == 0;
}

static inline
bool eqRect(const Rect* r1, const Rect* r2) {
	return r1->x == r2->x &&
		r1->y == r2->y &&
		r1->width == r2->width &&
		r1->height == r2->width;
}

static
void clearRect(Pixmap* fb, int pixSize, const Rect* rect) {
	Rect r = intersect(&fb->rect, rect);
	int clearOffset = r.x * pixSize;
	int clearSize = r.width * pixSize;
	int maxRow = r.y + r.height;

	for (int row = r.y; row < maxRow; ++row) {
		char* rowPtr = fb->data + row * fb->bytePerLine;
		memset(rowPtr + clearOffset, 0, clearSize);
	}
}

static
void drawPixmap(Pixmap* fb, int pixSize, const Pixmap* pixmap) {
	Rect r = intersect(&fb->rect, &pixmap->rect);
	int copySize = r.width * pixSize;
	int srcStartRow = r.y - pixmap->rect.y;
	int srcOffset = (r.x - pixmap->rect.x) * pixSize;
	int dstStartRow = r.y;
	int dstOffset = r.x * pixSize;

	for (int i = 0; i < r.height; ++i) {
		const char* srcRow = pixmap->data + (srcStartRow + i) * pixmap->bytePerLine;
		char* dstRow = fb->data + (dstStartRow + i) * fb->bytePerLine;
		memcpy(dstRow + dstOffset, srcRow + srcOffset, copySize);
	}
}

static
void drawPackedPixmapInsideFBU16(Pixmap* fb, const Pixmap* pixmap) {
	uint8_t* inPos = (uint8_t*)pixmap->data;
	uint8_t* inEnd = inPos + pixmap->dataSize;

	int lineNum = pixmap->rect.y;
	int outOffset = pixmap->rect.x*sizeof(uint16_t);
	uint16_t* outPos = (uint16_t*)(fb->data + outOffset);
	while (inPos != inEnd) {
		int pixCount = *inPos++;
		if (pixCount == 0) {
			// Line finished
			++lineNum;
			outPos = (uint16_t*)(fb->data + lineNum*fb->bytePerLine + outOffset);
			continue;
		}

		{
			uint16_t pix = *((uint16_t*)inPos);
			uint16_t* outEnd = outPos + pixCount;
			while (outPos != outEnd) {
				*outPos++ = pix;
			}
		}
		inPos += sizeof(uint16_t);
	}
}

static
void drawPackedPixmapNotInsideFBU16(Pixmap* fb, const Pixmap* pixmap) {
	int fbW = fb->rect.width;
	int fbH = fb->rect.height;
	uint8_t* inPos = (uint8_t*)pixmap->data;
	uint8_t* inEnd = inPos + pixmap->dataSize;

	// Part of the pixmap above the screen, skip it.
	int lineNum = pixmap->rect.y;
	while (lineNum < 0) {
		while (*inPos++ != 0) {
			inPos += sizeof(uint16_t);
		}
		++lineNum;
	}

	do {
		int x = pixmap->rect.x;

		int pixCount = *inPos++;
		uint16_t pix = *((uint16_t*)inPos);
		inPos += sizeof(uint16_t);
		if (x < 0) {
			// Part of pixmap to the left of the screen, skip it.
			for (;;) {
				int d = min(-x, pixCount);
				x += d;
				pixCount -= d;
				//printf("X=%d\n", x);
				if (x < 0) {
					pixCount = *inPos++;
					pix = *((uint16_t*)inPos);
					inPos += sizeof(uint16_t);
					continue;
				}
				break;
			}
		}

		{
			uint16_t* outPos = (uint16_t*)(fb->data + lineNum*fb->bytePerLine + x*sizeof(uint16_t));
			uint16_t* outEnd = (uint16_t*)(fb->data + lineNum*fb->bytePerLine + fbW*sizeof(uint16_t));
			for (;;) {
				pixCount = min(pixCount, outEnd - outPos);
				for (;pixCount > 0; --pixCount) {
					*outPos++ = pix;
				}

				if (outPos == outEnd) {
					// Skipping the rest of the line in pixmap
					while (*inPos++ != 0) {
						inPos += sizeof(uint16_t);
					}
					break;
				}

				pixCount = *inPos++;
				if (pixCount == 0) {
					break;
				}

				pix = *((uint16_t*)inPos);
				inPos += sizeof(uint16_t);
			}
		}

		++lineNum;
	} while (inPos != inEnd && lineNum < fbH);
}

static
void drawPackedPixmap(Pixmap* fb, int pixSize, const Pixmap* pixmap) {
	Rect intersectRect = intersect(&fb->rect, &pixmap->rect);
	if (isRectNull(&intersectRect)) {
		return;
	}

	if (eqRect(&intersectRect, &pixmap->rect)) {
		if (pixSize == 2)
			drawPackedPixmapInsideFBU16(fb, pixmap);
	} else {
		if (pixSize == 2)
			drawPackedPixmapNotInsideFBU16(fb, pixmap);
	}
}

static
void playCmds(Pixmap* fb, int pixSize, Cmd* cmds, int cmdCount) {
	int i;

	for (i = 0; i < cmdCount; ++i) {
		switch (cmds[i].code) {
		case ccClearRect:
			clearRect(fb, pixSize, &cmds[i].data.rect);
			break;
		case ccDrawPixmap:
			drawPixmap(fb, pixSize, &cmds[i].data.pixmap);
			break;
		case ccDrawPackedPixmap:
			drawPackedPixmap(fb, pixSize, &cmds[i].data.pixmap);
			break;
		default:
			break;
		}
	}
}
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"os"
	"syscall"
	"unsafe"

	drm "github.com/rmcsoft/godrm"
	"github.com/rmcsoft/godrm/mode"
)

const (
	startCmdCapacity = 256
)

type framebuffer struct {
	handle uint32
	id     uint32
	buf    []byte

	pixmap C.Pixmap
}

type kmsdrmPaintEngine struct {
	card    *os.File
	modeset mode.Modeset

	pixFormat PixelFormat
	pixSize   int

	framebuffers        []*framebuffer
	frontFrameBufferNum int

	isActive bool
	cmds     []C.Cmd
}

func (p *kmsdrmPaintEngine) GetWidth() int {
	return int(p.framebuffers[0].pixmap.rect.width)
}

func (p *kmsdrmPaintEngine) GetHeight() int {
	return int(p.framebuffers[0].pixmap.rect.height)
}

func (p *kmsdrmPaintEngine) Begin() error {
	if p.isActive {
		return errors.New("KMSDRMPaintEngine is already active")
	}

	p.isActive = true
	return nil
}

func (p *kmsdrmPaintEngine) Clear(rect image.Rectangle) error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	cmd := p.newCmd()
	cmd.code = C.ccClearRect
	cmdRect := (*C.Rect)(unsafe.Pointer(&cmd.data[0]))
	cmdRect.x = C.int(rect.Min.X)
	cmdRect.y = C.int(rect.Min.Y)
	cmdRect.width = C.int(rect.Dx())
	cmdRect.height = C.int(rect.Dy())
	return nil
}

func (p *kmsdrmPaintEngine) DrawPixmap(top image.Point, pixmap *Pixmap) error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	if p.pixFormat != pixmap.PixFormat {
		return errors.New("Pixmap has invalid pixel format")
	}

	rect := image.Rect(top.X, top.Y, top.X+pixmap.Width, top.Y+pixmap.Height)

	cmd := p.newCmd()
	cmd.code = C.ccDrawPixmap
	cmdPixmap := (*C.Pixmap)(unsafe.Pointer(&cmd.data[0]))
	cmdPixmap.rect.x = C.int(rect.Min.X)
	cmdPixmap.rect.y = C.int(rect.Min.Y)
	cmdPixmap.rect.width = C.int(rect.Dx())
	cmdPixmap.rect.height = C.int(rect.Dy())
	cmdPixmap.bytePerLine = C.int(pixmap.BytePerLine)
	cmdPixmap.data = (*C.char)(unsafe.Pointer(&pixmap.Data[0]))
	return nil
}

func (p *kmsdrmPaintEngine) DrawPackedPixmap(top image.Point, pixmap *PackedPixmap) error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	if p.pixFormat != pixmap.PixFormat {
		return errors.New("PackedPixmap has invalid pixel format")
	}

	rect := image.Rect(top.X, top.Y, top.X+pixmap.Width, top.Y+pixmap.Height)
	cmd := p.newCmd()
	cmd.code = C.ccDrawPackedPixmap
	cmdPixmap := (*C.Pixmap)(unsafe.Pointer(&cmd.data[0]))
	cmdPixmap.rect.x = C.int(rect.Min.X)
	cmdPixmap.rect.y = C.int(rect.Min.Y)
	cmdPixmap.rect.width = C.int(rect.Dx())
	cmdPixmap.rect.height = C.int(rect.Dy())
	cmdPixmap.data = (*C.char)(unsafe.Pointer(&pixmap.Data[0]))
	cmdPixmap.dataSize = C.int(len(pixmap.Data))
	return nil
}

func (p *kmsdrmPaintEngine) End() error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	frontFrameBuffer := p.framebuffers[p.frontFrameBufferNum]
	var cmds *C.Cmd
	if len(p.cmds) > 0 {
		cmds = &p.cmds[0]
	}
	C.playCmds(&frontFrameBuffer.pixmap, C.int(p.pixSize), cmds, C.int(len(p.cmds)))

	err := mode.SetCrtc(p.card, p.modeset.Crtc, frontFrameBuffer.id,
		0, 0, &p.modeset.Conn, 1, &p.modeset.Mode)

	p.cmds = p.cmds[:0]
	p.isActive = false
	p.frontFrameBufferNum = (p.frontFrameBufferNum + 1) % len(p.framebuffers)
	return err
}

// NewKMSDRMPaintEngine creates KMSDRMPaintEngine
func NewKMSDRMPaintEngine(cardNum int, pixFormat PixelFormat) (PaintEngine, error) {
	card, err := drm.OpenCard(cardNum)
	if err != nil {
		return nil, err
	}

	if !drm.HasDumbBuffer(card) {
		return nil, fmt.Errorf("drm device %v does not support dumb buffers", cardNum)
	}

	paintEngine := kmsdrmPaintEngine{
		card:      card,
		pixFormat: pixFormat,
		pixSize:   GetPixelSize(pixFormat),
		isActive:  false,
		cmds:      make([]C.Cmd, 0, startCmdCapacity),
	}

	simpleMSet, err := mode.NewSimpleModeset(card)
	if err != nil {
		return nil, err
	}

	if len(simpleMSet.Modesets) == 0 {
		return nil, errors.New("Modesets is empty")
	}

	paintEngine.modeset = simpleMSet.Modesets[0]
	paintEngine.framebuffers = []*framebuffer{}
	for i := 0; i < 2; i++ {
		framebuffer, err := paintEngine.createFramebuffer()
		if err != nil {
			return nil, err
		}
		paintEngine.framebuffers = append(paintEngine.framebuffers, framebuffer)
	}

	return &paintEngine, nil
}

func (p *kmsdrmPaintEngine) newCmd() *C.Cmd {
	p.cmds = append(p.cmds, C.Cmd{})
	return &p.cmds[len(p.cmds)-1]
}

func (p *kmsdrmPaintEngine) createFramebuffer() (*framebuffer, error) {

	fb := &framebuffer{}
	var err error

	defer func() {
		if err != nil {
			p.destroyFramebuffer(fb)
		}
	}()

	width := p.modeset.Width
	height := p.modeset.Height
	bpp := GetPixelSize(p.pixFormat) * 8
	depth := GetPixelDepth(p.pixFormat)

	fbInfo, err := mode.CreateFB(p.card, uint16(width), uint16(height), uint32(bpp))
	if err != nil {
		return nil, err
	}

	fb.handle = fbInfo.Handle
	fb.id, err = mode.AddFB(p.card, uint16(width), uint16(height),
		uint8(depth), uint8(bpp), fbInfo.Pitch, fb.handle)
	if err != nil {
		return nil, err
	}

	offset, err := mode.MapDumb(p.card, fb.handle)
	if err != nil {
		return nil, err
	}

	fb.buf, err = syscall.Mmap(int(p.card.Fd()), int64(offset), int(fbInfo.Size),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	fb.pixmap.rect.x = C.int(0)
	fb.pixmap.rect.y = C.int(0)
	fb.pixmap.rect.width = C.int(width)
	fb.pixmap.rect.height = C.int(height)
	fb.pixmap.data = (*C.char)(unsafe.Pointer(&fb.buf[0]))
	fb.pixmap.bytePerLine = C.int(fbInfo.Pitch)

	return fb, err
}

func (p *kmsdrmPaintEngine) destroyFramebuffer(fb *framebuffer) {
	if fb != nil && p.card != nil {
		if fb.id != 0 {
			mode.RmFB(p.card, fb.id)
			fb.id = 0
		}

		if fb.handle != 0 {
			mode.DestroyDumb(p.card, fb.handle)
			fb.handle = 0
		}

		if fb.buf != nil {
			syscall.Munmap(fb.buf)
			fb.buf = nil
		}
	}
}
