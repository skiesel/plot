package vgimg_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/skiesel/plot"
	"github.com/skiesel/plot/plotter"
	"github.com/skiesel/plot/vg"
	"github.com/skiesel/plot/vg/draw"
	"github.com/skiesel/plot/vg/vgimg"
)

func TestIssue179(t *testing.T) {
	scatter, err := plotter.NewScatter(plotter.XYs{{1, 1}, {0, 1}, {0, 0}})
	if err != nil {
		log.Fatal(err)
	}
	p, err := plot.New()
	if err != nil {
		log.Fatal(err)
	}
	p.Add(scatter)
	p.HideAxes()

	c := vgimg.JpegCanvas{Canvas: vgimg.New(5.08*vg.Centimeter, 5.08*vg.Centimeter)}
	p.Draw(draw.New(c))
	b := bytes.NewBuffer([]byte{})
	if _, err = c.WriteTo(b); err != nil {
		t.Error(err)
	}

	f, err := os.Open(filepath.Join("testdata", "issue179.jpg"))
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	want, err := ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(b.Bytes(), want) {
		t.Error("Image mismatch")
	}
}
