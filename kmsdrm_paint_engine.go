package chanim

/*
#cgo CFLAGS: -O3

#include <stdbool.h>
#include <stddef.h>
#include <string.h>
#include <stdlib.h>

typedef struct {
	int x;
	int y;
 	int width;
 	int height;
} Rect;

typedef struct {
	Rect rect;
	char* data;
	int bytePerLine;
} Pixmap;

typedef enum {
	ccClearRect,
	ccDrawPixmap
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
	ret.width = min(r1->x + r1->width, r1->x + r1->width) - ret.x;
	if (ret.width < 0)
		ret.width = 0;

	ret.y = max(r1->y, r2->y);
	ret.height = min(r1->y + r1->height, r2->y + r2->height) - ret.y;
	if (ret.height < 0)
		ret.height = 0;

		return ret;
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
	int srcStartRow = max(pixmap->rect.y, r.y);
	int srcOffset = max(pixmap->rect.x, r.x) * pixSize;
	int dstStartRow = max(fb->rect.y, r.y);
	int dstOffset = max(fb->rect.x, r.x) * pixSize;

	for (int i = 0; i < r.height; ++i) {
		char* srcRow = pixmap->data + (srcStartRow + i) * pixmap->bytePerLine;
		char* dstRow = fb->data + (dstStartRow + i) * fb->bytePerLine;
		memcpy(dstRow + dstOffset, srcRow + srcOffset, copySize);
	}
}

static inline
void clearPixmap(Pixmap* pixmap, int pixSize) {
	clearRect(pixmap, pixSize, &pixmap->rect);
}

void playCmds(Pixmap* fb, int pixSize, Cmd* cmds, int cmdCount) {
	int i;

	clearPixmap(fb, pixSize);
	for (i = 0; i < cmdCount; ++i) {
		switch (cmds[i].code) {
		case ccClearRect:
			clearRect(fb, pixSize, &cmds[i].data.rect);
			break;
		case ccDrawPixmap:
			drawPixmap(fb, pixSize, &cmds[i].data.pixmap);
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
	"image"
	"os"
	"unsafe"

	"github.com/NeowayLabs/drm"
	"github.com/NeowayLabs/drm/mode"
)

const (
	startCmdCapacity = 256
)

type framebuffer struct {
	fb     *mode.FB
	pixmap C.Pixmap
}

// KMSDRMPaintEngine is PaintEngine for kmsdrm
type KMSDRMPaintEngine struct {
	card      *os.File
	crtc      *mode.Crtc
	pixFormat PixelFormat
	pixSize   int
	viewport  image.Rectangle

	framebuffers      [2]framebuffer
	activeFramebuffer int

	isActive bool
	cmds     []C.Cmd
}

// Begin begins paint
func (p *KMSDRMPaintEngine) Begin() error {
	if p.isActive {
		return errors.New("KMSDRMPaintEngine is already active")
	}

	p.isActive = true
	return nil
}

// Clear clears the rectangle
func (p *KMSDRMPaintEngine) Clear(rect image.Rectangle) error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	cmd := p.newCmd()
	cmd.code = C.ccClearRect
	cmdRect := (*C.Rect)(unsafe.Pointer(&cmd.data[0]))
	cmdRect.x = C.int(rect.Min.X)
	cmdRect.y = C.int(rect.Min.Y)
	cmdRect.width = C.int(rect.Max.X - rect.Min.X)
	cmdRect.height = C.int(rect.Max.Y - rect.Min.Y)
	return nil
}

// DrawPixmap draws the Pixmap
func (p *KMSDRMPaintEngine) DrawPixmap(top image.Point, pixmap *Pixmap) error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	if p.pixFormat != pixmap.PixFormat {
		return errors.New("Pixmap has invalid pixel format")
	}

	cmd := p.newCmd()
	cmd.code = C.ccDrawPixmap
	cmdPixmap := (*C.Pixmap)(unsafe.Pointer(&cmd.data[0]))
	cmdPixmap.rect.x = C.int(top.X)
	cmdPixmap.rect.y = C.int(top.Y)
	cmdPixmap.rect.width = C.int(pixmap.Width)
	cmdPixmap.rect.height = C.int(pixmap.Height)
	cmdPixmap.bytePerLine = C.int(pixmap.BytePerLine)
	cmdPixmap.data = (*C.char)(unsafe.Pointer(&pixmap.Data[0]))
	return nil
}

// End ends paint
func (p *KMSDRMPaintEngine) End() error {
	if !p.isActive {
		return errors.New("KMSDRMPaintEngine is not active")
	}

	activeFb := &p.framebuffers[p.activeFramebuffer]
	var cmds *C.Cmd
	if len(p.cmds) > 0 {
		cmds = &p.cmds[0]
	}
	C.playCmds(&activeFb.pixmap, C.int(p.pixSize), cmds, C.int(len(p.cmds)))

	p.cmds = p.cmds[:0]
	p.isActive = false
	p.activeFramebuffer = (p.activeFramebuffer + 1) % len(p.framebuffers)
	return nil
}

// NewKMSDRMPaintEngine creates KMSDRMPaintEngine
func NewKMSDRMPaintEngine(cardNum int, pixFormat PixelFormat, viewport image.Rectangle) (PaintEngine, error) {
	card, err := drm.OpenCard(cardNum)
	if err != nil {
		return nil, err
	}

	paintEngine := KMSDRMPaintEngine{
		card:      card,
		pixFormat: pixFormat,
		pixSize:   GetPixelSize(pixFormat),
		viewport:  viewport,
		isActive:  false,
		cmds:      make([]C.Cmd, 0, startCmdCapacity),
	}

	return &paintEngine, nil
}

func (p *KMSDRMPaintEngine) newCmd() *C.Cmd {
	p.cmds = append(p.cmds)
	return &p.cmds[len(p.cmds)-1]
}

func createFramebuffer(card *os.File, dev *mode.Modeset) {
}
