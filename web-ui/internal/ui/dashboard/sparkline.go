package dashboard

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/internal/ui/style"
)

// chartHeight is the drawn area of every tile's chart, in points.
const chartHeight float32 = 64

// sparkline is a filled line chart of the last capacity samples, newest at
// the right edge. It draws itself into one canvas.Raster rather than out of
// hundreds of canvas.Line and canvas.Rectangle objects: the wasm driver
// uploads a raster as a single texture, whereas every separate object is
// its own draw call on every frame, five charts of them.
//
// The window is fixed: a chart with fewer samples than its capacity leaves
// the left part empty instead of stretching what it has, so the horizontal
// scale means the same five minutes from the first sample on.
type sparkline struct {
	widget.BaseWidget

	line     color.NRGBA
	capacity int
	// ceiling pins the top of the chart; zero means scale to the largest
	// sample in view.
	ceiling float64

	values []float64
	raster *canvas.Raster
}

func newSparkline(line color.NRGBA, capacity int, ceiling float64) *sparkline {
	s := &sparkline{line: line, capacity: max(capacity, 2), ceiling: ceiling}
	s.ExtendBaseWidget(s)
	s.raster = canvas.NewRaster(s.draw)
	return s
}

// Push appends a sample, dropping the oldest once the window is full.
func (s *sparkline) Push(v float64) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		v = 0
	}
	s.values = append(s.values, v)
	if len(s.values) > s.capacity {
		s.values = s.values[len(s.values)-s.capacity:]
	}
	s.raster.Refresh()
}

func (s *sparkline) CreateRenderer() fyne.WidgetRenderer {
	return &sparklineRenderer{s: s}
}

// draw rasterises the chart at the pixel size the driver asks for. Each
// pixel column between two samples is interpolated, filled down to the
// baseline in a translucent shade of the line colour, and capped with a
// two-pixel stroke.
func (s *sparkline) draw(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill(img, style.Background)
	if w < 2 || h < 2 || len(s.values) < 1 {
		return img
	}

	top := s.ceiling
	if top <= 0 {
		for _, v := range s.values {
			top = math.Max(top, v)
		}
		// Headroom so the peak does not touch the frame; a flat zero
		// series sits on the baseline rather than dividing by zero.
		top *= 1.15
		if top == 0 {
			top = 1
		}
	}

	// Vertical margins keep the stroke inside the raster at both extremes.
	const margin = 3.0
	usable := float64(h-1) - 2*margin
	yOf := func(v float64) float64 {
		return margin + usable*(1-math.Min(v/top, 1))
	}

	// Samples are spaced so that a full window spans the whole width; a
	// partial one occupies the right-hand end.
	step := float64(w-1) / float64(s.capacity-1)
	n := len(s.values)
	x0 := float64(w-1) - step*float64(n-1)

	area := style.WithAlpha(s.line, 0x38)
	for i := 0; i < n; i++ {
		xa := x0 + step*float64(i)
		ya := yOf(s.values[i])
		xb, yb := xa, ya
		if i+1 < n {
			xb = x0 + step*float64(i+1)
			yb = yOf(s.values[i+1])
		}
		for x := int(math.Round(xa)); x <= int(math.Round(xb)) && x < w; x++ {
			t := 0.0
			if xb > xa {
				t = (float64(x) - xa) / (xb - xa)
			}
			y := ya + (yb-ya)*t
			yi := int(math.Round(y))
			for py := yi; py < h; py++ {
				img.SetNRGBA(x, py, area)
			}
			img.SetNRGBA(x, yi, s.line)
			if yi+1 < h {
				img.SetNRGBA(x, yi+1, s.line)
			}
		}
	}
	return img
}

func fill(img *image.NRGBA, c color.NRGBA) {
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

type sparklineRenderer struct {
	s *sparkline
}

func (r *sparklineRenderer) Layout(size fyne.Size) { r.s.raster.Resize(size) }
func (r *sparklineRenderer) MinSize() fyne.Size    { return fyne.NewSize(80, chartHeight) }
func (r *sparklineRenderer) Refresh()              { r.s.raster.Refresh() }
func (r *sparklineRenderer) Destroy()              {}

func (r *sparklineRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.s.raster}
}
