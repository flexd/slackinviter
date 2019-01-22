package badge

import (
	"html/template"
	"io"
	"sync"

	"github.com/golang/freetype/truetype"
	"github.com/narqo/go-badge/fonts"
	"golang.org/x/image/font"
)

type badge struct {
	Subject string
	Status  string
	Color   Color
	Bounds  bounds
}

type bounds struct {
	// SubjectDx is the width of subject string of the badge.
	SubjectDx float64
	SubjectX  float64
	// StatusDx is the width of status string of the badge.
	StatusDx  float64
	StatusX   float64
}

func (b bounds) Dx() float64 {
	return b.SubjectDx + b.StatusDx
}

type badgeDrawer struct {
	fd    *font.Drawer
	tmpl  *template.Template
	mutex *sync.Mutex
}

func (d *badgeDrawer) Render(subject, status string, color Color, w io.Writer) error {
	d.mutex.Lock()
	subjectDx := d.measureString(subject)
	statusDx := d.measureString(status)
	d.mutex.Unlock()

	bdg := badge{
		Subject: subject,
		Status:  status,
		Color:   color,
		Bounds: bounds{
			SubjectDx: subjectDx,
			SubjectX:  subjectDx / 2.0 + 1,
			StatusDx:  statusDx,
			StatusX:   subjectDx + statusDx / 2.0 - 1,
		},
	}
	return d.tmpl.Execute(w, bdg)
}

// shield.io uses Verdana.ttf to measure text width with an extra 10px.
// As we use Vera.ttf, we have to tune this value a little.
const extraDx = 13

func (d *badgeDrawer) measureString(s string) float64 {
	sm := d.fd.MeasureString(s)
	// this 64 is weird but it's the way I've found how to convert fixed.Int26_6 to float64
	return float64(sm) / 64 + extraDx
}

// Render renders a badge of the given color, with given subject and status to w.
func Render(subject, status string, color Color, w io.Writer) error {
	return drawer.Render(subject, status, color, w)
}

const (
	dpi = 72
	fontsize = 11
)

var drawer *badgeDrawer

func init() {
	drawer = &badgeDrawer{
		fd:   mustNewFontDrawer(fontsize, dpi),
		tmpl: template.Must(template.New("flat-template").Parse(flatTemplate)),
		mutex: &sync.Mutex{},
	}
}

func mustNewFontDrawer(size, dpi float64) *font.Drawer {
	ttf, err := truetype.Parse(fonts.VeraSans)
	if err != nil {
		panic(err)
	}
	return &font.Drawer{
		Face: truetype.NewFace(ttf, &truetype.Options{
			Size:    size,
			DPI:     dpi,
			Hinting: font.HintingFull,
		}),
	}
}
