package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	overturemaps "github.com/ringsaturn/overture-maps"
	"github.com/ringsaturn/polyf"
	"github.com/ringsaturn/polyf/integration/featurecollection"
	"github.com/tidwall/geojson/geometry"
	"github.com/tidwall/rtree"
	"go.uber.org/zap"
)

//go:embed template/*
var f embed.FS

// country
// county
// state
// region
// province
// district
// city
// town
// village
// hamlet
// borough
// suburb
// neighborhood
// municipality

var localityTypeOrder map[string]int = map[string]int{
	"country":      0,
	"county":       1,
	"state":        2,
	"region":       3,
	"province":     4,
	"district":     5,
	"city":         6,
	"town":         7,
	"village":      8,
	"hamlet":       9,
	"borough":      10,
	"suburb":       11,
	"neighborhood": 12,
	"municipality": 13,
}

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

var localityIDToOrbMultiPolygon map[string]orb.MultiPolygon = map[string]orb.MultiPolygon{}

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
			localityIDToOrbMultiPolygon[newV.LocalityID] = g
		case orb.Polygon:
			poly := convertPolygon(g)
			res = append(res, &polyf.Item[overturemaps.LocalityAreaRowProperties]{
				V:    *newV,
				Poly: poly,
			})
			localityIDToOrbMultiPolygon[newV.LocalityID] = orb.MultiPolygon{g}
		}
	}
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

type Request struct {
	Lng       float64 `query:"lng"`
	Lat       float64 `query:"lat"`
	Debug     bool    `query:"debug"`
	DebugFull bool    `query:"debug_full"`
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

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	h := server.New(
		server.WithHostPorts("localhost:5070"),
	)
	h.SetHTMLTemplate(template.Must(template.New("").ParseFS(f, "template/*.html")))
	h.GET("/reverse", func(c context.Context, ctx *app.RequestContext) {
		req := &Request{}
		if err := ctx.Bind(req); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		if !req.Debug {
			res, err := finder.FindAll(req.Lng, req.Lat)
			if err != nil {
				ctx.String(http.StatusInternalServerError, err.Error())
				return
			}
			result := make([]*overturemaps.LocalityRowProperties, 0)
			for _, r := range res {
				if p, ok := localityFinder.M[r.LocalityID]; ok {
					result = append(result, p)
				}
			}
			sort.Slice(result, func(i, j int) bool {
				if localityTypeOrder[result[i].LocalityType] < localityTypeOrder[result[j].LocalityType] {
					return true
				}
				return localityTypeOrder[result[i].ContextID] == localityTypeOrder[result[j].ID]
			})
			ctx.JSON(http.StatusOK, result)
			return
		}

		boundary := &featurecollection.BoundaryFile[*overturemaps.LocalityRowProperties]{
			Features: make([]*featurecollection.Feature[*overturemaps.LocalityRowProperties], 0),
		}
		point := geometry.Point{
			X: req.Lng,
			Y: req.Lat,
		}
		for _, item := range finder.Items {
			if item.Poly.ContainsPoint(point) {
				if p, ok := localityFinder.M[item.V.LocalityID]; ok {
					boundary.Features = append(boundary.Features, &featurecollection.Feature[*overturemaps.LocalityRowProperties]{
						Properties: p,
						Type:       "Feature",
						Geometry: struct {
							Coordinates interface{} "json:\"coordinates\""
							Type        string      "json:\"type\""
						}{
							Type:        "MultiPolygon",
							Coordinates: localityIDToOrbMultiPolygon[item.V.LocalityID],
						},
					})
				}
			}
		}

		sort.Slice(boundary.Features, func(i, j int) bool {
			// return localityTypeOrder[boundary.Features[i].Properties.LocalityType] < localityTypeOrder[boundary.Features[j].Properties.LocalityType]
			if localityTypeOrder[boundary.Features[i].Properties.LocalityType] < localityTypeOrder[boundary.Features[j].Properties.LocalityType] {
				return true
			}
			return localityTypeOrder[boundary.Features[i].Properties.ContextID] != localityTypeOrder[boundary.Features[j].Properties.ID]
		})

		if !req.DebugFull {
			// keep the latest one
			if len(boundary.Features) > 1 {
				boundary.Features = boundary.Features[len(boundary.Features)-1:]
			}
		}

		ctx.JSON(http.StatusOK, utils.H{
			"type":     "FeatureCollection",
			"features": boundary.Features,
		})
	})

	h.GET("/", func(c context.Context, ctx *app.RequestContext) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	_ = h.Run()

	// // idx := 0
	// for _, value := range localityIDToOrbMultiPolygon {
	// 	b, _ := json.Marshal(value)
	// 	fmt.Println(string(b))
	// 	break
	// }

	// fmt.Println("done")
	// time.Sleep(time.Minute)
}
