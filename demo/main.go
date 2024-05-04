package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	overturemaps "github.com/ringsaturn/overture-maps"
	"github.com/ringsaturn/polyf"
	"github.com/tidwall/geojson/geometry"
	"github.com/tidwall/rtree"
	"go.uber.org/zap"
)

var logger *zap.Logger = func() *zap.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return l
}()

func convertPolygon(rawPoly orb.Polygon) *geometry.Poly {
	exterior := []geometry.Point{}
	holes := [][]geometry.Point{}
	for i, geopoly := range rawPoly {
		if i == 0 {
			for _, c := range geopoly {
				exterior = append(exterior, geometry.Point{X: c.X(), Y: c.Y()})
			}
			continue
		}
		holepoints := []geometry.Point{}

		for _, c := range geopoly {
			holepoints = append(holepoints, geometry.Point{X: c.X(), Y: c.Y()})
		}
		holes = append(holes, holepoints)
	}
	poly := geometry.NewPoly(exterior, holes, nil)
	return poly
}

func convertMultiPolygon(rawMulti orb.MultiPolygon) []*geometry.Poly {
	polys := []*geometry.Poly{}
	for _, rawPoly := range rawMulti {
		polys = append(polys, convertPolygon(rawPoly))
	}
	return polys
}

func loadLocalityAreaAsPolyfItem(fp string) ([]*polyf.Item[overturemaps.LocalityAreaRowProperties], error) {
	logger.Debug("loadLocalityAreaAsPolyfItem", zap.String("fp", fp))
	f, err := os.ReadFile(fp)
	if err != nil {
		panic(err)
	}
	rows, err := overturemaps.ReadFile[overturemaps.LocalityAreaRow](io.NewSectionReader(bytes.NewReader(f), 0, int64(len(f))))
	if err != nil {
		panic(err)
	}

	res := make([]*polyf.Item[overturemaps.LocalityAreaRowProperties], 0)

	typeCounter := map[string]int{}

	for _, row := range rows {
		geometry, err := wkb.Unmarshal(row.Geometry)
		if err != nil {
			panic(err)
		}
		newV := row.ToProperties()
		switch g := geometry.(type) {
		case orb.MultiPolygon:
			polys := convertMultiPolygon(g)

			for _, poly := range polys {
				res = append(res, &polyf.Item[overturemaps.LocalityAreaRowProperties]{
					V:    *newV,
					Poly: poly,
				})
			}
			t := fmt.Sprintf("%T", g)
			if _, ok := typeCounter[t]; !ok {
				typeCounter[t] = 0
			}
			typeCounter[t]++
		case orb.Polygon:
			poly := convertPolygon(g)
			res = append(res, &polyf.Item[overturemaps.LocalityAreaRowProperties]{
				V:    *newV,
				Poly: poly,
			})
			t := fmt.Sprintf("%T", g)
			if _, ok := typeCounter[t]; !ok {
				typeCounter[t] = 0
			}
			typeCounter[t]++
		default:
			t := fmt.Sprintf("%T", g)
			if _, ok := typeCounter[t]; !ok {
				typeCounter[t] = 0
			}
			typeCounter[t]++
		}
	}
	logger.Debug("loadLocalityAreaAsPolyfItem", zap.Int("len", len(res)), zap.Any("typeCounter", typeCounter))
	return res, nil
}

func setupLocalityAreaFinder(dir string) (*polyf.F[overturemaps.LocalityAreaRowProperties], error) {
	items := []*polyf.Item[overturemaps.LocalityAreaRowProperties]{}

	visit := func(path string, d os.DirEntry, err error) error {
		defer runtime.GC()
		logger.Debug("visit", zap.String("path", path))
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".parquet" {
			return nil
		}
		_items, _err := loadLocalityAreaAsPolyfItem(path)
		if _err != nil {
			return _err
		}
		logger.Debug("visit", zap.Int("len", len(_items)))
		items = append(items, _items...)
		return nil
	}

	err := filepath.WalkDir(dir, visit)
	if err != nil {
		return nil, err
	}

	finder := &polyf.F[overturemaps.LocalityAreaRowProperties]{
		Items: items,
	}
	return finder, nil
}

type LocalityFinder struct {
	Tr *rtree.RTreeG[*overturemaps.LocalityRowProperties]
	M  map[string]*overturemaps.LocalityRowProperties // LocalityID -> LocalityRow
}

func setupLocalityItems(fp string) ([]overturemaps.LocalityRow, error) {
	f, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	return overturemaps.ReadFile[overturemaps.LocalityRow](io.NewSectionReader(bytes.NewReader(f), 0, int64(len(f))))
}

func setupLocalityFinder(dir string) (*LocalityFinder, error) {
	items := []overturemaps.LocalityRow{}

	visit := func(path string, d os.DirEntry, err error) error {
		defer runtime.GC()
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".parquet" {
			return nil
		}
		_items, _err := setupLocalityItems(path)
		if _err != nil {
			return _err
		}
		items = append(items, _items...)
		return nil
	}

	err := filepath.WalkDir(dir, visit)
	if err != nil {
		return nil, err
	}

	tr := &rtree.RTreeG[*overturemaps.LocalityRowProperties]{}
	m := map[string]*overturemaps.LocalityRowProperties{}

	for _, item := range items {
		p := item.ToProperties()
		m[item.ID] = p
		tr.Insert(
			[2]float64{float64(item.BBox.XMin), float64(item.BBox.YMin)},
			[2]float64{float64(item.BBox.XMax), float64(item.BBox.YMax)},
			p,
		)
	}

	return &LocalityFinder{
		Tr: tr,
		M:  m,
	}, nil

}

func main() {
	dir := "themes-2024M04/type=locality_area"

	finder, err := setupLocalityAreaFinder(dir)
	if err != nil {
		panic(err)
	}

	localityFinder, err := setupLocalityFinder("themes-2024M04/type=locality")
	if err != nil {
		panic(err)
	}

	inputLng := -74.006
	inputLat := 40.7128

	res, err := finder.FindAll(inputLng, inputLat)
	if err != nil {
		panic(err)
	}
	for _, r := range res {
		if p, ok := localityFinder.M[r.LocalityID]; ok {
			logger.Info("FindAll", zap.String("LocalityID", r.LocalityID), zap.String("Name", p.Names.Primary))
		} else {
			logger.Info("FindAll", zap.String("LocalityID", r.LocalityID), zap.String("Name", "Unknown"))
		}
	}
	localityFinder.Tr.Search(
		[2]float64{inputLng - 0.003, inputLat - 0.003},
		[2]float64{inputLng + 0.003, inputLat + 0.003},
		func(min, max [2]float64, data *overturemaps.LocalityRowProperties) bool {
			fmt.Println(data.Names.Primary)
			return true
		},
	)
	fmt.Println("done")
	time.Sleep(time.Minute)
}
